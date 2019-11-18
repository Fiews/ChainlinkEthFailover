package main

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

type Strategy string

const RoundRobin Strategy = "roundrobin"
const PrimaryAsync Strategy = "primary-async"
const PrimaryInstant Strategy = "primary-instant"

func (service *Service) FindEndpoint() *Endpoint {
	for _, e := range service.Endpoints {
		e.SetShouldDisconnect(false)
	}

	if len(service.Endpoints) == 1 {
		return service.Endpoints[0]
	}

	switch service.Config.Strategy {
	case RoundRobin:
		return service.pickRoundRobin()
	case PrimaryInstant:
		return service.pickPrimaryInstant()
	case PrimaryAsync:
		return service.pickPrimaryAsync()
	}

	return service.Endpoints[0]
}

func (service *Service) pickRoundRobin() *Endpoint {
	for _, e := range service.Endpoints {
		if e.OfflineSince == nil {
			return e
		}
	}
	var oldestCheck *Endpoint
	for _, e := range service.Endpoints {
		if oldestCheck == nil || e.OfflineSince.Before(*oldestCheck.OfflineSince) {
			oldestCheck = e
		}
	}
	// Force offline since to now, so it isn't picked first next iteration
	oldestCheck.OfflineSince = nil
	return oldestCheck
}

func (service *Service) pickPrimaryInstant() *Endpoint {
	var leastAttempts *Endpoint
	for _, e := range service.Endpoints {
		if e.OfflineSince == nil {
			return e
		}
		if e.FailedAttempts < service.Config.MaxAttempts {
			return e
		}
		if leastAttempts == nil || e.FailedAttempts < leastAttempts.FailedAttempts {
			leastAttempts = e
		}
	}
	return leastAttempts
}

func (service *Service) pickPrimaryAsync() *Endpoint {
	// Suggestion for strategy to not stick to the first endpoint,
	// but to follow Primary-Instant
	/*primary := service.pickPrimaryInstant()
	if primary.OfflineSince == nil {
		return primary
	}

	secondary := service.pickRoundRobin()
	if primary.Id == secondary.Id {
		secondary = service.pickRoundRobin()
	}*/
	primary := service.pickRoundRobin()
	if primary.OfflineSince == nil {
		return primary
	}

	secondary := service.pickRoundRobin()
	if primary.Id == secondary.Id {
		return secondary
	}

	go func() {
		canConnect := func() bool {
			var dialer = websocket.Dialer{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, _, err := dialer.DialContext(ctx, primary.Url, nil)
			if err != nil {
				return false
			}
			conn.Close()
			return true
		}
		primary.IncrementFailedAttempts()

		fmt.Println("Checking endpoint in background:", primary.Url)

		connected := false
		for !connected {
			if secondary.ShouldDisconnect {
				return
			}

			fmt.Println("Waiting", service.Config.ReconnectTimeout.Seconds(), "seconds before reconnect attempt to", primary.Url)
			time.Sleep(service.Config.ReconnectTimeout)
			connected = canConnect()
		}

		fmt.Println("Successfully opened connection to", primary.Url)
		primary.OfflineSince = nil
		primary.FailedAttempts = 0
		primary.LastHeader = nil
		secondary.ShouldDisconnect = true
	}()

	return secondary
}
