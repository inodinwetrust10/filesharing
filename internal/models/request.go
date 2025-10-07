// Package models
package models

type Request struct {
	Type     string `json:"type"`
	Content  any    `json:"content"`
	Username string `json:"username"`
}
