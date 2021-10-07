package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	NEW_LINE = []byte{'\n'}
	SPACE   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  2048,
	WriteBufferSize: 2048,
}

const (
	// Time allowed to write a message to the peer.
	WRITE_WAIT = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	PONG_WAIT = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	PING_PERIOD = (PONG_WAIT * 9) / 10

	// Maximum message size allowed from peer.
	MAX_MESSAGE_SIZE = 1024
)

const (
	ARD_AGENT_CONNECTED 	= 1
	ARD_AGENT_DISCONNECTED 	= 2
)

type ARDAgent struct {
	Hub 			*ARDAgentHub
	Connection		*websocket.Conn
	SignalMessage	chan []byte
}

type ARDAgentMessage struct {
	ARDAgent	*ARDAgent
	Data		[]byte
}

func (agent *ARDAgent)ReadDispatch() {
	defer func() {
		agent.Hub.SignalConnectionState <- SignalConnectionStateMeta {
			Agent: agent,
			State: ARD_AGENT_DISCONNECTED,
		}
		agent.Connection.Close()
		fmt.Println("Connection to agent " + agent.Connection.RemoteAddr().String() + " closed.")
	}()

	agent.Connection.SetReadLimit(MAX_MESSAGE_SIZE)
	agent.Connection.SetReadDeadline(time.Now().Add(PONG_WAIT))
	agent.Connection.SetPongHandler(func(string) error { 
		agent.Connection.SetReadDeadline(time.Now().Add(PONG_WAIT));
		return nil 
	})
	for {
		_, message, err := agent.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(message)
		agent.Hub.SignalMessage <- ARDAgentMessage{
			ARDAgent: agent,
			Data: message,
		}
	}
}

func (agent *ARDAgent)WriteDispatch() {
	ticker := time.NewTicker(PING_PERIOD)
	defer func() {
		ticker.Stop()
		agent.Connection.Close()
	}()
	for {
		select {
		case message, ok := <-agent.SignalMessage:
			agent.Connection.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if !ok {
				// The hub closed the channel.
				agent.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				fmt.Println("Failed to read message from channel. Internal Error !")
				return
			}

			w, err := agent.Connection.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Println("Failed to get writer to send message.")
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(agent.SignalMessage)
			for i := 0; i < n; i++ {
				w.Write(<-agent.SignalMessage)
			}

			if err := w.Close(); err != nil {
				fmt.Println("Failed to close writer.")
				return
			}
		case <-ticker.C:
			agent.Connection.SetWriteDeadline(time.Now().Add(WRITE_WAIT))
			if err := agent.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("Failed to send ping message.")
				return
			}
		}
	}
}

func (agent *ARDAgent)ArdAgentGetAddressString() string {
	return agent.Connection.RemoteAddr().String()
}

func ARDAgentServe(hub *ARDAgentHub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	agent := &ARDAgent{Hub: hub, Connection: conn, SignalMessage: make(chan []byte, 1024)}
	agent.Hub.SignalConnectionState <- SignalConnectionStateMeta {
		Agent: agent,
		State: ARD_AGENT_CONNECTED,
	}

	fmt.Println("New Connection from agent " + agent.Connection.RemoteAddr().String())

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go agent.WriteDispatch()
	go agent.ReadDispatch()
}