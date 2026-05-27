package handlers

import (
	"archive/zip"
	"backend-app/config"
	"backend-app/internal/share"
	"backend-app/internal/tunnel"
	"backend-app/internal/types"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ok",
		"tunnel_active": tunnel.IsRunning(),
	})
}

func HandleShares(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		type CreateReq struct {
			Paths    []string `json:"paths"`
			Label    string   `json:"label"`
			Password string   `json:"password"`
			Note     string   `json:"note"`

			IsInternet bool `json:"isInternet"`
			IsLAN      bool `json:"isLAN"`
		}
		var req CreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if len(req.Paths) == 0 {
			http.Error(w, "No files specified", http.StatusBadRequest)
			return
		}

		baseURL := ""
		if req.IsInternet {
			t, err := tunnel.Acquire()
			if err != nil {
				http.Error(w, "Failed to start tunnel", http.StatusInternalServerError)
				return
			}

			baseURL = t.URL
		}

		if baseURL == "" {
			baseURL = "http://localhost:" + config.ServerPort
		}
		localIP := GetLocalIP()
		localBaseURL := fmt.Sprintf("http://%s:%s", localIP, config.ServerPort)

		s, err := share.Create(req.Paths, req.Label, baseURL, localBaseURL, req.Password, req.Note, req.IsInternet, req.IsLAN)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := types.CreateResp{
			Token:            s.Token,
			DownloadURL:      s.PublicDownloadURL,
			LocalDownloadURL: s.LocalDownloadURL,
			ShareID:          s.ID,
			PublicBaseURL:    baseURL,

			IsInternet: s.IsInternet,
			IsLAN:      s.IsLAN,

			Password: s.Password,
			Note:     s.Note,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	} else if r.Method == http.MethodGet {
		list := share.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"shares":        list,
			"tunnel_active": tunnel.IsRunning(),
		})
		return
	} else if r.Method == http.MethodDelete {
		token := r.URL.Query().Get("token")
		if token == "" {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) > 3 {
				token = parts[3]
			}
		}

		if token == "" {
			http.Error(w, "Missing token", http.StatusBadRequest)
			return
		}

		s, exists := share.Get(token)
		if !exists {
			http.Error(w, "Share not found", http.StatusNotFound)
			return
		}

		deleted := share.Delete(token)
		if !deleted {
			http.Error(w, "Share not found", http.StatusNotFound)
			return
		}

		if s.IsInternet {
			_ = tunnel.Release()
		}

		share.TransfersMu.Lock()
		for key := range share.BlockedIPs {
			if strings.HasPrefix(key, token+"_") {
				delete(share.BlockedIPs, key)
			}
		}
		share.TransfersMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "deleted"}`))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func HandleTransfers(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		ip := r.URL.Query().Get("ip")
		token := r.URL.Query().Get("token")
		if ip == "" || token == "" {
			http.Error(w, "Missing ip or token", http.StatusBadRequest)
			return
		}

		share.TransfersMu.Lock()
		share.BlockedIPs[token+"_"+ip] = true
		for _, conn := range share.ActiveConnections {
			if conn.Token == token && conn.IP == ip {
				conn.Cancelled = true
				conn.Cancel()
			}
		}
		share.TransfersMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"cancelled"}`))
		return
	}

	share.TransfersMu.RLock()
	defer share.TransfersMu.RUnlock()

	// Group by token
	grouped := map[string][]*types.TransferStats{}

	// Let's iterate over share.CumulativeBytes to find all unique token + ip pairs
	for key, bytes := range share.CumulativeBytes {
		parts := strings.Split(key, "_")
		if len(parts) < 2 {
			continue
		}
		token := parts[0]
		ip := parts[1]

		// Get total size of the share
		var totalSize int64
		if s, exists := share.Get(token); exists {
			totalSize = s.TotalSize
		} else {
			continue // share no longer exists
		}

		// Calculate current speed and activity of this IP
		var speed float64
		isActive := false
		var lastUpdated time.Time
		var startTime time.Time

		for _, conn := range share.ActiveConnections {
			if conn.Token == token && conn.IP == ip {
				speed += conn.Speed
				isActive = true
				if startTime.IsZero() || conn.StartTime.Before(startTime) {
					startTime = conn.StartTime
				}
				if conn.LastUpdated.After(lastUpdated) {
					lastUpdated = conn.LastUpdated
				}
			}
		}

		if startTime.IsZero() {
			startTime = time.Now()
		}
		if lastUpdated.IsZero() {
			lastUpdated = time.Now()
		}

		stats := &types.TransferStats{
			Token:        token,
			SessionID:    ip, // Use IP as the SessionID so the client uses it as connection key
			BytesWritten: bytes,
			TotalBytes:   totalSize,
			Speed:        speed,
			IsActive:     isActive,
			StartTime:    startTime,
			LastUpdated:  lastUpdated,
		}

		if isActive {
			grouped[token] = append(grouped[token], stats)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(grouped)
}

func HandleDownload(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/share/")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	s, exists := share.Get(token)
	if !exists {
		http.NotFound(w, r)
		return
	}

	if len(s.FilePaths) == 0 {
		http.Error(w, "No files in share", http.StatusInternalServerError)
		return
	}

	clientIP := GetClientIP(r)

	share.TransfersMu.RLock()
	isBlocked := share.BlockedIPs[token+"_"+clientIP]
	share.TransfersMu.RUnlock()

	if isBlocked {
		http.Error(w, "Access denied: disconnected by host", http.StatusForbidden)
		return
	}

	// Verify password authentication
	isAuthenticated := false
	if s.Password == "" {
		isAuthenticated = true
	} else {
		// 1. Check query parameter
		if r.URL.Query().Get("pwd") == s.Password {
			isAuthenticated = true
		} else {
			// 2. Check cookie
			cookie, err := r.Cookie("justsent_auth_" + token)
			if err == nil && cookie.Value == "1" {
				isAuthenticated = true
			}
		}
	}

	if !isAuthenticated {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")

			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "Invalid form submission data.",
				})
				return
			}

			pwd := r.FormValue("password")
			if pwd == s.Password {
				http.SetCookie(w, &http.Cookie{
					Name:     "justsent_auth_" + token,
					Value:    "1",
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					MaxAge:   86400, // 24 hours
				})
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
				})
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Incorrect password. Please try again.",
			})
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		GetPasswordPageTemplate().Execute(w, types.PageData{
			Label:              s.Label,
			FileCount:          len(s.FilePaths),
			TotalSizeFormatted: FormatBytes(s.TotalSize),
			Note:               s.Note,
		})
		return
	}

	// Create cancelable context for this connection
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	r = r.WithContext(ctx)

	// Initialize active transfer connection
	sessionID := fmt.Sprintf("%s_%d", token, time.Now().UnixNano())
	share.TransfersMu.Lock()
	share.ActiveConnections[sessionID] = &types.Connection{
		SessionID:    sessionID,
		Token:        token,
		IP:           clientIP,
		BytesWritten: 0,
		Speed:        0,
		StartTime:    time.Now(),
		LastUpdated:  time.Now(),
		Cancel:       cancel,
	}
	share.TransfersMu.Unlock()

	defer func() {
		share.TransfersMu.Lock()
		delete(share.ActiveConnections, sessionID)
		share.TransfersMu.Unlock()
	}()

	pw := &progressWriter{
		w:         w,
		token:     token,
		sessionID: sessionID,
		lastTime:  time.Now(),
		lastBytes: 0,
	}

	if len(s.FilePaths) == 1 {
		filePath := s.FilePaths[0]
		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "Failed to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Failed to stat file", http.StatusInternalServerError)
			return
		}

		pw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Name()))
		pw.Header().Set("Content-Type", "application/octet-stream")
		pw.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		http.ServeContent(pw, r, fileInfo.Name(), fileInfo.ModTime(), file)
		return
	}

	// Serve zipped files on-the-fly
	pw.Header().Set("Content-Type", "application/zip")
	pw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", s.Label))

	zipWriter := zip.NewWriter(pw)
	defer zipWriter.Close()

	for _, path := range s.FilePaths {
		func() {
			file, err := os.Open(path)
			if err != nil {
				return
			}
			defer file.Close()

			fileInfo, err := file.Stat()
			if err != nil {
				return
			}

			header, err := zip.FileInfoHeader(fileInfo)
			if err != nil {
				return
			}

			header.Name = filepath.Base(path)
			header.Method = zip.Deflate

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return
			}

			_, _ = io.Copy(writer, file)
		}()
	}
}
