package main

import (
	"github.com/google/uuid"
	"sync"
	"time"
)

type Endpoint struct {
	Id               uuid.UUID
	Lock             *sync.RWMutex
	Url              string
	OfflineSince     *time.Time
	LastHeader       *time.Time
	FailedAttempts   int
	ShouldDisconnect bool
}

func CreateEndpoint(url string) *Endpoint {
	return &Endpoint{
		Id:   uuid.New(),
		Lock: &sync.RWMutex{},
		Url:  url,
	}
}

func (e *Endpoint) IncrementFailedAttempts() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	e.FailedAttempts++
	if e.OfflineSince == nil {
		now := time.Now()
		e.OfflineSince = &now
	}
}

func (e *Endpoint) UpdateLastHeader() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	now := time.Now()
	e.LastHeader = &now
	// If there are new headers, endpoint is considered healthy
	e.OfflineSince = nil
	e.FailedAttempts = 0
}

func (e *Endpoint) SetOffline() {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	now := time.Now()
	e.OfflineSince = &now
}

func (e *Endpoint) SetShouldDisconnect(b bool) {
	e.Lock.Lock()
	defer e.Lock.Unlock()
	e.ShouldDisconnect = b
}
