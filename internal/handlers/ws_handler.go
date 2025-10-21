// Package handlers
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/inodinwetrust/filesharing/internal/models"
)

const buffer = 1024 * 128

var upgrader = websocket.Upgrader{
	ReadBufferSize:  buffer,
	WriteBufferSize: buffer,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	ActiveUsers = make(models.ActiveUsers)
	mtx         sync.Mutex
)

func Upgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	go handleConnections(conn)
}

func handleConnections(conn *websocket.Conn) {
	var connectionRequest models.Request
	err := conn.ReadJSON(&connectionRequest)
	if err != nil || connectionRequest.Type != "connectionRequest" {
		log.Printf("Bad connection request received. Error: %v", err)
		conn.WriteJSON(map[string]string{"type": "bad request"})
		conn.Close()
		return
	}

	username := connectionRequest.Username
	mtx.Lock()
	if _, ok := ActiveUsers[username]; ok {
		mtx.Unlock()
		log.Printf("Username %s is already taken", username)
		conn.WriteJSON(map[string]string{"error": "username already taken", "type": "connectionError"})
		conn.Close()
		return
	}

	ActiveUsers[username] = conn
	mtx.Unlock()

	conn.SetReadLimit(buffer)
	log.Printf("User %s connected.", username)
	conn.WriteJSON(map[string]string{"type": "connectionSuccess"})
	time.Sleep(100 * time.Millisecond)
	BroadcastAllActiveUsers()

	defer func() {
		mtx.Lock()
		delete(ActiveUsers, username)
		mtx.Unlock()
		conn.Close()
		log.Printf("User %s disconnected.", username)
		BroadcastAllActiveUsers()
	}()

	var transferTargetUser string
	var chunkCount int
	var bytesTransferred int64

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from %s: %v. Disconnecting.", username, err)
			break
		}

		if messageType == websocket.TextMessage {
			var msg models.FileTransferRequest
			if err := json.Unmarshal(p, &msg); err != nil {
				log.Printf("Error unmarshalling JSON from %s: %v", username, err)
				continue
			}

			if msg.To == "" {
				log.Printf("Received text message from %s without a 'to' recipient.", username)
				continue
			}

			mtx.Lock()
			targetConn, ok := ActiveUsers[msg.To]
			mtx.Unlock()

			if !ok {
				log.Printf("User %s tried to send to offline user %s", username, msg.To)
				conn.WriteJSON(map[string]string{"type": "error", "message": "User " + msg.To + " is not online."})
				continue
			}

			msg.From = username

			switch msg.Message {
			case "file-start":
				log.Printf("File transfer starting: %s -> %s (File: %v)", username, msg.To, msg.Payload)
				transferTargetUser = msg.To
				chunkCount = 0
				bytesTransferred = 0

				if err := targetConn.WriteJSON(msg); err != nil {
					log.Printf("Failed to forward file-start from %s to %s: %v", username, msg.To, err)
				}

			case "file-end":
				log.Printf("File transfer complete: %s -> %s (%d chunks, %d bytes)",
					username, transferTargetUser, chunkCount, bytesTransferred)

				if err := targetConn.WriteJSON(msg); err != nil {
					log.Printf("Failed to forward file-end from %s to %s: %v", username, msg.To, err)
				}

				transferTargetUser = ""
				chunkCount = 0
				bytesTransferred = 0

			default:
				if err := targetConn.WriteJSON(msg); err != nil {
					log.Printf("Failed to forward message from %s to %s: %v", username, msg.To, err)
				}
			}

		} else if messageType == websocket.BinaryMessage {
			if transferTargetUser == "" {
				log.Printf("Received an unexpected binary message from %s.", username)
				continue
			}

			mtx.Lock()
			targetConn, ok := ActiveUsers[transferTargetUser]
			mtx.Unlock()

			if !ok {
				log.Printf("Transfer target %s for sender %s disconnected.", transferTargetUser, username)
				conn.WriteJSON(map[string]string{"type": "error", "message": "User " + transferTargetUser + " disconnected during transfer."})
				transferTargetUser = ""
				chunkCount = 0
				bytesTransferred = 0
				continue
			}

			chunkCount++
			bytesTransferred += int64(len(p))

			targetConn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if err := targetConn.WriteMessage(websocket.BinaryMessage, p); err != nil {
				log.Printf("Failed to forward binary chunk #%d from %s to %s: %v",
					chunkCount, username, transferTargetUser, err)
				transferTargetUser = ""
				chunkCount = 0
				bytesTransferred = 0
			} else {
				if chunkCount%50 == 0 {
					log.Printf("Progress: %s -> %s (chunk %d, %d bytes)",
						username, transferTargetUser, chunkCount, bytesTransferred)
				}
			}
			targetConn.SetWriteDeadline(time.Time{})
		}
	}
}

func BroadcastAllActiveUsers() {
	mtx.Lock()
	defer mtx.Unlock()

	allUsernames := make([]string, 0, len(ActiveUsers))
	for username := range ActiveUsers {
		allUsernames = append(allUsernames, username)
	}

	for recipientUsername, recipientConn := range ActiveUsers {
		otherUsers := make([]string, 0, len(allUsernames)-1)
		for _, username := range allUsernames {
			if username != recipientUsername {
				otherUsers = append(otherUsers, username)
			}
		}

		broadcast := models.Broadcast{
			Type:    "allonlineusers",
			Content: otherUsers,
		}

		if err := recipientConn.WriteJSON(broadcast); err != nil {
			log.Printf("Error broadcasting to %s, removing connection: %v", recipientUsername, err)
			recipientConn.Close()
			delete(ActiveUsers, recipientUsername)
		}
	}
}
