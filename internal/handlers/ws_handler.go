// Package handlers
package handlers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/inodinwetrust/filesharing/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var ActiveUsers = make(models.ActiveUsers)

func Upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	go handleConnections(conn)
}

func handleConnections(conn *websocket.Conn) {
	var connectionRequest models.Request

	err := conn.ReadJSON(&connectionRequest)
	log.Printf("Received connection request data: %+v", connectionRequest)
	if err != nil || connectionRequest.Type != "connectionRequest" {
		log.Printf("Bad request received. Error: %v", err)
		conn.WriteJSON(map[string]string{
			"type": "bad request",
		})
		conn.Close()
		return
	}
	username := connectionRequest.Username
	mtx.Lock()
	if _, ok := ActiveUsers[username]; ok {
		log.Printf("username %s is already taken", username)
		mtx.Unlock()
		conn.WriteJSON(map[string]string{
			"type": "username already taken",
		})
		conn.Close()
		return
	}
	ActiveUsers[connectionRequest.Username] = conn
	mtx.Unlock()
	conn.WriteJSON(map[string]string{
		"type": "connectionSuccess",
	})
	time.Sleep(100 * time.Millisecond)
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
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.Type {
		case "sendFile":
			{
				SendFiles(conn, &msg)
			}
		}
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
