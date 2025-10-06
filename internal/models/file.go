package models

type FileTransferRequest struct {
	Type   string
	Size   int
	SentTo string // username
}
