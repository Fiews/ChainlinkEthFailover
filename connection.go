package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type JsonrpcMessage struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// Utility function to quickly jsonify the jsonrpcMessage to be sent over Websockets.
func (msg *JsonrpcMessage) Json() []byte {
	b, _ := json.Marshal(msg)
	return b
}

type Status int

const (
	CLOSED Status = iota
)

type Connection struct {
	endpoint      *Endpoint
	connected     *time.Time
	validated     *time.Time
	client        *WsConn
	eth           *WsConn
	busy          *uint32
	connection    chan Status
	expectedClose bool
}

type WsConn struct {
	Ws     *websocket.Conn
	Reader *sync.Mutex
	Writer *sync.Mutex
}

func (service *Service) initConnection(conn *websocket.Conn, r *http.Request) (*Connection, error) {
	connected := time.Now()
	var zero uint32 = 0
	con := &Connection{
		connected: &connected,
		client: &WsConn{
			Ws:     conn,
			Reader: &sync.Mutex{},
			Writer: &sync.Mutex{},
		},
		busy:       &zero,
		connection: make(chan Status),
		endpoint:   service.FindEndpoint(),
	}

	ethConFunc := func() (*websocket.Conn, error) {
		var dialer = websocket.Dialer{}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, _, err := dialer.DialContext(ctx, con.endpoint.Url, nil)
		return conn, err
	}

	ethCon, err := ethConFunc()
	if err != nil {
		con.endpoint.IncrementFailedAttempts()
		conn.Close()
		return nil, err
	}

	con.eth = &WsConn{
		Ws:     ethCon,
		Reader: &sync.Mutex{},
		Writer: &sync.Mutex{},
	}

	return con, nil
}

// Close disconnects the connection.
// It will expect several gouroutines to call Close, so it will get an atomic lock on the actual disconnecting.
func (conn *Connection) Close() {
	if !atomic.CompareAndSwapUint32(conn.busy, 0, 1) {
		return
	}

	conn.endpoint.SetShouldDisconnect(true)

	fmt.Println("Disconnecting from", conn.endpoint.Url)
	close(conn.connection)

	if conn.client != nil && conn.client.Ws != nil {
		conn.client.Ws.Close()
	}

	if conn.eth != nil && conn.eth.Ws != nil {
		conn.eth.Ws.Close()
	}
}

func (conn *Connection) handleConn(bhnTimeout time.Duration) {
	go conn.watchIncoming()
	go conn.watchOutgoing()

	fmt.Println("Successfully connected to", conn.endpoint.Url)

	started := time.Now()

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		if conn.endpoint.ShouldDisconnect {
			return
		}
		select {
		case <-conn.connection:
			return
		case <-t.C:
			lastHeader := started
			if conn.endpoint.LastHeader != nil {
				lastHeader = *conn.endpoint.LastHeader
			}

			if time.Now().Sub(lastHeader) > bhnTimeout {
				fmt.Println("No block header received within timeout")
				conn.endpoint.IncrementFailedAttempts()
				conn.endpoint.SetOffline()
				conn.Close()
				return
			}
		}
	}
}

func (conn *Connection) watchIncoming() {
	defer conn.Close()

	for {
		// Read message from the CL node
		_, rawMsg, err := conn.client.Ws.ReadMessage()
		if err != nil {
			conn.expectedClose = true
			return
		}

		var msg JsonrpcMessage
		err = json.Unmarshal(rawMsg, &msg)
		if err == nil {
			go conn.incomingMsg(&msg)
		}
	}
}

type bhnParams struct {
	Subscription string `json:"subscription"`
	Result       struct {
		Difficulty string `json:"difficulty"`
		Timestamp  string `json:"timestamp"`
		Miner      string `json:"miner"`
		ParentHash string `json:"parentHash"`
	} `json:"result"`
}

func isBlockHeaderNotification(data []byte) bool {
	var bhn bhnParams
	err := json.Unmarshal(data, &bhn)
	if err != nil {
		return false
	}

	if len(bhn.Result.Difficulty) == 0 || len(bhn.Result.ParentHash) == 0 {
		return false
	}

	return true
}

func (conn *Connection) watchOutgoing() {
	defer conn.Close()

	for {
		// Read message from ETH node
		_, rawMsg, err := conn.eth.Ws.ReadMessage()
		if err != nil {
			if !conn.expectedClose {
				now := time.Now()
				conn.endpoint.OfflineSince = &now
			}
			return
		}

		var msg JsonrpcMessage
		err = json.Unmarshal(rawMsg, &msg)
		if err == nil {
			if msg.Method == "eth_subscription" && isBlockHeaderNotification(msg.Params) {
				conn.endpoint.UpdateLastHeader()
			}
			go conn.outgoingMsg(&msg)
		}
	}
}

func (conn *Connection) outgoingMsg(msg *JsonrpcMessage) {
	conn.client.Writer.Lock()
	defer conn.client.Writer.Unlock()
	err := conn.client.Ws.WriteMessage(websocket.TextMessage, msg.Json())
	if err != nil {
		// Could not write in this connection.
		// Assume that it is closed
		conn.Close()
	}
}

func (conn *Connection) incomingMsg(msg *JsonrpcMessage) {
	conn.client.Writer.Lock()
	defer conn.client.Writer.Unlock()
	err := conn.eth.Ws.WriteMessage(websocket.TextMessage, msg.Json())
	if err != nil {
		// Could not write in this connection.
		// Assume that it is closed
		conn.Close()
	}
}
