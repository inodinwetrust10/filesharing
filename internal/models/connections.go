package models

import "github.com/gorilla/websocket"

type ActiveUsers map[string]*websocket.Conn

type Broadcast struct {
	Type    string   `json:"type"`
	Content []string `json:"content"`
}
