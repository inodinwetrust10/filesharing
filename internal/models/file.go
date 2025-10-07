package models

type FileTransferRequest struct {
	Type           string `json:"type"`
	SentTo         string `json:"recipientUsername"`
	FileName       string `json:"fileName"`
	SenderUsername string `json:"senderUsername"`
}

type FileNotification struct {
	Type    string `json:"type"`
	Content any    `json:""`
}
