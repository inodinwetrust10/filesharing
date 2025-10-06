package models

import "github.com/gorilla/websocket"

type ActiveUsers map[string]*websocket.Conn

type Broadcast struct {
	Type    string
	Content []string
}

type FormatError struct {
	Message string
}
