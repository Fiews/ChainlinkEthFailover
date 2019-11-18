package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
)

type Service struct {
	Config    Config
	Endpoints []*Endpoint
}

func startService(config Config, endpointsString []string) {
	endpoints := make([]*Endpoint, len(endpointsString))
	for i, e := range endpointsString {
		fmt.Println("Adding endpoint", e)
		endpoints[i] = CreateEndpoint(e)
	}

	service := Service{
		Config:    config,
		Endpoints: endpoints,
	}

	http.HandleFunc("/", service.WsRequest)

	fmt.Println("Listening on port", service.Config.Port)
	http.ListenAndServe(fmt.Sprintf(":%v", service.Config.Port), nil)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (service Service) WsRequest(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	connection, err := service.initConnection(conn, r)
	if err != nil {
		fmt.Println(err)
		return
	}

	connection.handleConn(service.Config.HeaderTimeout)
}
