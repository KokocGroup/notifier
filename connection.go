package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"strings"
	"time"
)

// initMessage is a parsed value of incoming JSON
type initMessage struct {
	UID       int    `json:"uid"`
	Timestamp int64  `json:"timestamp"`
	Checksum  string `json:"checksum"`
}

type errorMessage struct {
	Error string `json:"error"`
}

// Validates checksum
func (m *initMessage) validate() error {
	h := md5.New()
	_, err := fmt.Fprintf(h, "%d%d%s", m.UID, m.Timestamp, *secret)
	if err != nil {
		return errors.New("cannot calculate checksum")
	}
	hash := h.Sum([]byte{})
	if fmt.Sprintf("%x", hash) != strings.ToLower(m.Checksum) {
		return errors.New("checksum is invalid")
	}
	return nil
}

// UserConnection represents user ws-connection and his UID
type UserConnection struct {
	UID int
	ws  *websocket.Conn
}

// NewUserConnection constructor for UserConnection
func NewUserConnection(ws *websocket.Conn) *UserConnection {
	return &UserConnection{ws: ws}
}

// Send method deliver message on websocket
func (u *UserConnection) Send(m Message) error {
	return u.send([]byte(m.Message))
}

func (u *UserConnection) send(m []byte) error {
	return u.ws.WriteMessage(websocket.TextMessage, m)
}

// Listen method listens ws-connection and tries to get user UID
func (u *UserConnection) Listen() {
	defer func() {
		u.ws.Close()
		registry.Unregister(u)
	}()

	for {
		_, rawMessage, err := u.ws.ReadMessage()
		if err != nil {
			break
		}
		var msg = initMessage{}
		err = json.Unmarshal(rawMessage, &msg)
		if err != nil {
			errorMsgB, _ := json.Marshal(&errorMessage{Error: err.Error()})
			err = u.send(errorMsgB)
			continue
		}
		// If checksum is one day old - ignore attempt
		if msg.Timestamp < time.Now().Unix()-24*60*60 {
			errorMsgB, _ := json.Marshal(&errorMessage{Error: "timestamp in initial sequence is too old or not presented"})
			err = u.send(errorMsgB)
			continue
		}
		err = msg.validate()
		if err != nil {
			errorMsgB, _ := json.Marshal(&errorMessage{Error: err.Error()})
			err = u.send(errorMsgB)
			continue
		}
		u.UID = msg.UID
		registry.Register(u)
	}
}
