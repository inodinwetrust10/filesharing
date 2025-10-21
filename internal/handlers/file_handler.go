package handlers

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/inodinwetrust/filesharing/internal/models"
)

func SendFiles(conn *websocket.Conn, msg *models.FileTransferRequest) {
	senderUsername := msg.From
	recipientUsername := msg.To
	mtx.Lock()
	recipientConn, ok := ActiveUsers[msg.To]
	mtx.Unlock()

	if !ok {
		log.Printf("recipient is disconnected or not found %s", msg.To)
		return
	}
	notification := models.FileNotification{
		Type: "incomingFile",
		Content: map[string]string{
			"fileName": msg.Payload.Name,
			"sender":   senderUsername,
		},
	}
	err := recipientConn.WriteJSON(&notification)
	if err != nil {
		log.Print("error sending the request")
	}
	log.Print("Notified about sending file")

	for {
		msgType, chunks, err := conn.ReadMessage()
		if err != nil {
			log.Printf("finished reading message from %s", senderUsername)
			break
		}

		if msgType == websocket.BinaryMessage {
			if err := recipientConn.WriteMessage(websocket.BinaryMessage, chunks); err != nil {
				log.Printf("failed to relay chunks to %s", recipientUsername)
				break
			}
		}
	}
}
