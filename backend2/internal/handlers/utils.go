package handlers

import (
	"backend-app/internal/share"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"
)

type progressWriter struct {
	w         http.ResponseWriter
	token     string
	sessionID string
	lastTime  time.Time
	lastBytes int64
}

func (pw *progressWriter) Header() http.Header {
	return pw.w.Header()
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	// Check if the share has been deleted/stopped
	if _, exists := share.Get(pw.token); !exists {
		return 0, fmt.Errorf("share aborted by host")
	}

	// Check if the connection has been cancelled by user disconnect request
	share.TransfersMu.RLock()
	conn, exists := share.ActiveConnections[pw.sessionID]
	var isCancelled bool
	if exists && conn.Cancelled {
		isCancelled = true
	}
	share.TransfersMu.RUnlock()

	if isCancelled {
		return 0, fmt.Errorf("connection disconnected by user")
	}

	n, err := pw.w.Write(p)
	if n > 0 {
		share.TransfersMu.Lock()
		if conn, exists := share.ActiveConnections[pw.sessionID]; exists {
			conn.BytesWritten += int64(n)
			share.CumulativeBytes[conn.Token+"_"+conn.IP] += int64(n)

			now := time.Now()
			elapsed := now.Sub(pw.lastTime).Seconds()
			if elapsed >= 0.5 { // calculate speed every 500ms
				writtenInWindow := conn.BytesWritten - pw.lastBytes
				conn.Speed = float64(writtenInWindow) / elapsed
				pw.lastTime = now
				pw.lastBytes = conn.BytesWritten
				conn.LastUpdated = now
			}
		}
		share.TransfersMu.Unlock()
	}
	return n, err
}

func (pw *progressWriter) WriteHeader(statusCode int) {
	pw.w.WriteHeader(statusCode)
}

func GetClientIP(r *http.Request) string {
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func GetPasswordPageTemplate() *template.Template {
	const passwordPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Secure Share | JustSent</title>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg: #F9F8F6;
            --card-bg: #FFFFFF;
            --text-primary: #1F1F1B;
            --text-secondary: #6E6A63;
            --text-muted: #9E9990;
            --accent: #B8642A;
            --accent-hover: #9E5422;
            --accent-soft: #F3E4D8;
            --border: #E7E1D8;
            --error: #EF4444;
            --error-bg: #FEE2E2;
            --success: #10B981;
            --success-bg: #E6F4EA;
            --shadow: 0 4px 20px rgba(110, 106, 99, 0.05);
            --transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background-color: var(--bg);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 24px;
        }

        .container {
            width: 100%;
            max-width: 440px;
            background: var(--card-bg);
            border: 1px solid var(--border);
            border-radius: 16px;
            padding: 40px;
            box-shadow: var(--shadow);
            transition: var(--transition);
            position: relative;
            overflow: hidden;
        }

        .logo-row {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 32px;
            justify-content: center;
        }

        .logo-icon {
            width: 28px;
            height: 28px;
            background: var(--accent);
            color: #FFFFFF;
            border-radius: 6px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 800;
            font-size: 16px;
        }

        .logo-text {
            font-size: 16px;
            font-weight: 700;
            letter-spacing: -0.02em;
            color: var(--text-primary);
        }

        h1 {
            font-size: 20px;
            font-weight: 700;
            margin-bottom: 8px;
            color: var(--text-primary);
            letter-spacing: -0.02em;
            text-align: center;
        }

        .subtitle {
            font-size: 13px;
            color: var(--text-secondary);
            line-height: 1.5;
            text-align: center;
            margin-bottom: 32px;
        }

        .file-info-box {
            display: flex;
            align-items: center;
            gap: 12px;
            background-color: #FAF9F6;
            border: 1px solid var(--border);
            border-radius: 10px;
            padding: 14px 16px;
            margin-bottom: 24px;
        }

        .file-icon {
            width: 38px;
            height: 38px;
            background: var(--accent-soft);
            color: var(--accent);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 16px;
            flex-shrink: 0;
        }

        .file-meta {
            min-width: 0;
            flex: 1;
        }

        .file-name {
            font-size: 14px;
            font-weight: 600;
            color: var(--text-primary);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-bottom: 2px;
        }

        .file-size {
            font-size: 12px;
            color: var(--text-secondary);
        }

        .note-box {
            background-color: rgba(184, 100, 42, 0.04);
            border: 1px dashed rgba(184, 100, 42, 0.2);
            border-radius: 10px;
            padding: 14px 16px;
            margin-bottom: 24px;
            font-size: 13px;
            line-height: 1.5;
            color: var(--accent-hover);
        }

        .note-title {
            font-weight: 700;
            font-size: 11px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 4px;
            display: flex;
            align-items: center;
            gap: 4px;
        }

        .error-message {
            background-color: var(--error-bg);
            border: 1px solid rgba(239, 68, 68, 0.2);
            color: var(--error);
            padding: 10px 14px;
            border-radius: 8px;
            font-size: 13px;
            margin-bottom: 20px;
            display: none;
            align-items: center;
            gap: 8px;
        }

        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-4px); }
            75% { transform: translateX(4px); }
        }

        .input-group {
            margin-bottom: 20px;
        }

        .input-label {
            display: block;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-secondary);
            margin-bottom: 8px;
        }

        .input-wrapper {
            position: relative;
        }

        .input-field {
            width: 100%;
            background: #FFFFFF;
            border: 1px solid var(--border);
            color: var(--text-primary);
            padding: 12px 14px;
            border-radius: 8px;
            font-size: 14px;
            font-family: inherit;
            transition: var(--transition);
            outline: none;
        }

        .input-field:focus {
            border-color: var(--accent);
            box-shadow: 0 0 0 3px rgba(184, 100, 42, 0.15);
        }

        .submit-btn {
            width: 100%;
            background: var(--accent);
            color: #FFFFFF;
            border: none;
            padding: 12px;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: var(--transition);
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }

        .submit-btn:hover {
            background: var(--accent-hover);
        }

        .submit-btn:disabled {
            background: var(--text-muted);
            cursor: not-allowed;
            opacity: 0.7;
        }

        .spinner {
            animation: rotate 1s linear infinite;
            width: 16px;
            height: 16px;
        }

        @keyframes rotate {
            100% { transform: rotate(360deg); }
        }

        .footer {
            margin-top: 32px;
            text-align: center;
            font-size: 11px;
            color: var(--text-muted);
        }

        .footer a {
            color: var(--text-secondary);
            text-decoration: none;
            font-weight: 500;
        }

        .footer a:hover {
            text-decoration: underline;
        }

        /* Success & Animation States */
        .container.success-state {
            transform: scale(0.97);
            opacity: 0;
        }

        .success-wrapper {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            text-align: center;
            padding: 20px 0;
            opacity: 0;
            transform: translateY(10px);
            animation: slideUpFade 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
        }

        @keyframes slideUpFade {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .success-icon-container {
            display: flex;
            justify-content: center;
            margin-bottom: 20px;
        }

        .checkmark {
            width: 48px;
            height: 48px;
            border-radius: 50%;
            display: block;
            stroke-width: 2.5;
            stroke: var(--success);
            stroke-miterlimit: 10;
            box-shadow: inset 0px 0px 0px var(--success);
            animation: fillCheck 0.4s ease-in-out 0.4s forwards, scaleCheck 0.3s ease-in-out 0.9s forwards;
        }

        .checkmark__circle {
            stroke-dasharray: 166;
            stroke-dashoffset: 166;
            stroke-width: 2.5;
            stroke-miterlimit: 10;
            stroke: var(--success);
            fill: none;
            animation: strokeCheck 0.6s cubic-bezier(0.65, 0, 0.45, 1) forwards;
        }

        .checkmark__check {
            transform-origin: 50% 50%;
            stroke-dasharray: 48;
            stroke-dashoffset: 48;
            animation: strokeCheck 0.3s cubic-bezier(0.65, 0, 0.45, 1) 0.6s forwards;
        }

        @keyframes strokeCheck {
            100% { stroke-dashoffset: 0; }
        }

        @keyframes scaleCheck {
            0%, 100% { transform: none; }
            50% { transform: scale3d(1.08, 1.08, 1); }
        }

        @keyframes fillCheck {
            100% { box-shadow: inset 0px 0px 0px 30px var(--success-bg); }
        }

        .success-title {
            font-size: 18px;
            font-weight: 700;
            color: var(--text-primary);
            margin-bottom: 6px;
        }

        .success-message {
            font-size: 13px;
            color: var(--text-secondary);
            margin-bottom: 24px;
        }

        .loader-bar {
            width: 120px;
            height: 4px;
            background-color: var(--border);
            border-radius: 2px;
            overflow: hidden;
            position: relative;
        }

        .loader-progress {
            width: 40%;
            height: 100%;
            background-color: var(--accent);
            border-radius: 2px;
            position: absolute;
            animation: loadingMove 1.2s infinite ease-in-out;
        }

        @keyframes loadingMove {
            0% { left: -40%; }
            50% { left: 100%; }
            100% { left: 100%; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div id="form-content">
            <div class="logo-row">
                <div class="logo-icon">J</div>
                <div class="logo-text">JustSent</div>
            </div>

            <h1>Protected Share</h1>
            <p class="subtitle">This share requires a password to unlock and start the download.</p>

            <div class="file-info-box">
                <div class="file-icon">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>
                </div>
                <div class="file-meta">
                    <div class="file-name" title="{{.Label}}">{{.Label}}</div>
                    <div class="file-size">{{.FileCount}} files • {{.TotalSizeFormatted}}</div>
                </div>
            </div>

            {{if .Note}}
            <div class="note-box">
                <div class="note-title">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>
                    Note from sender
                </div>
                <div>{{.Note}}</div>
            </div>
            {{end}}

            <div class="error-message"></div>

            <form method="POST" action="">
                <div class="input-group">
                    <label class="input-label" for="password">Password</label>
                    <div class="input-wrapper">
                        <input class="input-field" type="password" id="password" name="password" placeholder="Enter share password" required autofocus>
                    </div>
                </div>

                <button type="submit" class="submit-btn">
                    <span>Unlock & Download</span>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
                </button>
            </form>

            <div class="footer">
                <p>Secured with local file sharing by <a href="https://github.com" target="_blank">JustSent</a></p>
            </div>
        </div>
    </div>

    <script>
        const form = document.querySelector('form');
        const container = document.querySelector('.container');
        const formContent = document.getElementById('form-content');
        const submitBtn = document.querySelector('.submit-btn');

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            hideError();
            const password = document.getElementById('password').value;
            
            // Set loading state
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<svg class="spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="2" x2="12" y2="6"></line><line x1="12" y1="18" x2="12" y2="22"></line><line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line><line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line><line x1="2" y1="12" x2="6" y2="12"></line><line x1="18" y1="12" x2="22" y2="12"></line><line x1="4.93" y1="19.07" x2="7.76" y2="16.24"></line><line x1="16.24" y1="7.76" x2="19.07" y2="4.93"></line></svg><span>Verifying...</span>';
            
            try {
                const res = await fetch(window.location.pathname, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: new URLSearchParams({ password })
                });
                
                const data = await res.json();
                
                if (data.success) {
                    // Smooth transition to success
                    container.classList.add('success-state');
                    
                    setTimeout(() => {
                        // Replace content
                        container.innerHTML = '<div class="success-wrapper"><div class="success-icon-container"><svg class="checkmark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 52 52"><circle class="checkmark__circle" cx="26" cy="26" r="25" fill="none"/><path class="checkmark__check" fill="none" d="M14.1 27.2l7.1 7.2 16.7-16.8"/></svg></div><h2 class="success-title">Verified</h2><p class="success-message">Connecting secure download stream...</p><div class="loader-bar"><div class="loader-progress"></div></div></div>';
                        // Remove fadeout class to trigger entry animation of success state
                        container.classList.remove('success-state');
                        
                        // Trigger actual native browser download via background link click
                        setTimeout(() => {
                            const dlLink = document.createElement('a');
                            dlLink.href = window.location.pathname;
                            document.body.appendChild(dlLink);
                            dlLink.click();
                            document.body.removeChild(dlLink);

                            // Update message and hide loader bar
                            const msgEl = document.querySelector('.success-message');
                            if (msgEl) msgEl.innerText = 'Download started successfully.';
                            const barEl = document.querySelector('.loader-bar');
                            if (barEl) barEl.style.display = 'none';
                        }, 1200);
                    }, 250);
                } else {
                    submitBtn.disabled = false;
                    submitBtn.innerHTML = '<span>Unlock & Download</span><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>';
                    showError(data.error || "Incorrect password.");
                }
            } catch (err) {
                submitBtn.disabled = false;
                submitBtn.innerHTML = '<span>Unlock & Download</span><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>';
                showError("Connection error. Please try again.");
            }
        });

        function showError(msg) {
            const errEl = document.querySelector('.error-message');
            errEl.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg><span>' + msg + '</span>';
            errEl.style.display = 'flex';
            errEl.style.animation = 'none';
            errEl.offsetHeight; // trigger reflow
            errEl.style.animation = 'shake 0.4s ease-in-out';
        }

        function hideError() {
            const errEl = document.querySelector('.error-message');
            errEl.style.display = 'none';
        }
    </script>
</body>
</html>`

	var pwdTmpl = template.Must(template.New("password").Parse(passwordPageTemplate))
	return pwdTmpl
}
