// Package handlers
package handlers

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/inodinwetrust/filesharing/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
var ActiveUsers models.ActiveUsers

func Upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	go handleConnections(conn, r)
}

func handleConnections(conn *websocket.Conn, r *http.Request) {
	var body models.ConnectionRequest
	if err := conn.ReadJSON(&body); err != nil || body.Type != "Request" {
		log.Print("bad request")
		conn.Close()
		return
	}
	username := body.Username

	mtx.Lock()
	if _, ok := ActiveUsers[username]; ok {
		log.Printf("username %s is already taken", username)
		mtx.Unlock()
		conn.Close()
		return
	}
	mtx.Lock()
	ActiveUsers[body.Username] = conn
	mtx.Unlock()
	BroadcastAllActiveUsers()

	defer func() {
		mtx.Lock()
		delete(ActiveUsers, username)
		mtx.Unlock()
		conn.Close()
		BroadcastAllActiveUsers()
	}()

	for {
		var msg models.FileTransferRequest
	}
}

var mtx sync.Mutex

func BroadcastAllActiveUsers() {
	connectionsToSend := make(map[string]*websocket.Conn)

	allOnlineUsers := make([]string, 0)

	mtx.Lock()
	for username, conn := range ActiveUsers {
		allOnlineUsers = append(allOnlineUsers, username)
		connectionsToSend[username] = conn
	}
	mtx.Unlock()
	broadcast := models.Broadcast{
		Type:    "allonlineusers",
		Content: allOnlineUsers,
	}

	for username, conn := range connectionsToSend {
		if err := conn.WriteJSON(broadcast); err != nil {
			log.Printf("error sending online users to %s, disconnecting", username)
			conn.Close()
			mtx.Lock()
			delete(ActiveUsers, username)
			mtx.Unlock()
		}
	}
}
