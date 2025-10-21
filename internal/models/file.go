package models

type FileMetadata struct {
	Name string `json:"name"`
	Type string `json:"string"`
	Size int64  `json:"size"`
}

type FileTransferRequest struct {
	From    string        `json:"from,omitempty"`
	To      string        `json:"to"`
	Message string        `json:"message"`
	Payload *FileMetadata `json:"payload,omitempty"`
}

type FileNotification struct {
	Type    string `json:"type"`
	Content any    `json:""`
}
