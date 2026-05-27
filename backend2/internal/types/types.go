package types

import (
	"context"
	"net/http"
	"time"
)

type Share struct {
	ID                int
	Token             string
	FilePaths         []string
	Label             string
	PrimaryName       string
	PublicDownloadURL string
	LocalDownloadURL  string
	IsInternet        bool
	IsLAN             bool
	FileCount         int
	TotalSize         int64
	Password          string
	Note              string
	Downloads         int
	CreatedAt         time.Time
	ExpiresAt         *time.Time
}

type CreateResp struct {
	Token            string `json:"token"`
	DownloadURL      string `json:"download_url"`
	LocalDownloadURL string `json:"local_download_url"`
	ShareID          int    `json:"share_id"`
	PublicBaseURL    string `json:"public_base_url"`
	IsInternet       bool   `json:"is_internet"`
	IsLAN            bool   `json:"is_lan"`
	Password         string `json:"password,omitempty"`
	Note             string `json:"note,omitempty"`
}

type TransferStats struct {
	Token        string    `json:"token"`
	SessionID    string    `json:"session_id"`
	BytesWritten int64     `json:"bytes_written"`
	TotalBytes   int64     `json:"total_bytes"`
	Speed        float64   `json:"speed"` // bytes per second
	IsActive     bool      `json:"is_active"`
	StartTime    time.Time `json:"-"`
	LastUpdated  time.Time `json:"-"`
}

type Connection struct {
	SessionID    string
	Token        string
	IP           string
	BytesWritten int64
	Speed        float64
	StartTime    time.Time
	LastUpdated  time.Time
	Cancel       context.CancelFunc
	Cancelled    bool
}

type ProgressWriter struct {
	w         http.ResponseWriter
	token     string
	sessionID string
	lastTime  time.Time
	lastBytes int64
}

type PageData struct {
	Label              string
	FileCount          int
	TotalSizeFormatted string
	Note               string
	Error              string
}
