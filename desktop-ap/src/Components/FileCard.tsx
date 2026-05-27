import { useState, useEffect } from "react";
import { formatFileSize, getExtension } from "../utils/fileFormatting";
import { SettingsIcon } from "../assets/icons";
import { type TransferStats, disconnectDownloader } from "../lib/backend";

export type StatefulFile = {
  id: string;
  name: string;
  size?: number;
  path?: string;
  isSharing: boolean;
  shareLink: string | null;
  localShareLink: string | null;
  shareInternet: boolean;
  shareNearby: boolean;
  shareError: string | null;
  shareCreating: boolean;
  isActionsOpen: boolean;
  isDownloading?: boolean;
  bytesWritten?: number;
  speed?: number;
  isCompleted?: boolean;
  activeDownloads?: TransferStats[];
  passwordProtected: boolean;
  passwordValue: string;
  noteValue: string;
};

type FileCardProps = {
  file: StatefulFile;
  onToggleActions: () => void;
  onRemoveFile: () => void;
  onStartSharing: () => void | Promise<void>;
  onStopSharing: () => void | Promise<void>;
  onToggleShareInternet: () => void;
  onToggleShareNearby: () => void;
  onTogglePasswordProtected: () => void;
  onChangePasswordValue: (val: string) => void;
  onChangeNoteValue: (val: string) => void;
};

const getExtClass = (name: string) => {
  const ext = getExtension(name).toLowerCase();
  if (ext === "pdf") return "ext-pdf";
  if (["zip", "rar", "tar", "gz", "7z"].includes(ext)) return "ext-zip";
  if (["png", "jpg", "jpeg", "gif", "svg", "webp", "heic"].includes(ext)) return "ext-img";
  if (["mp4", "mov", "avi", "mkv", "webm"].includes(ext)) return "ext-video";
  if (["mp3", "wav", "m4a", "flac", "ogg"].includes(ext)) return "ext-audio";
  return "ext-default";
};

const getDownloaderEta = (bytesWritten: number, totalBytes: number, speed: number) => {
  if (speed <= 0) return "Calculating...";
  const remaining = totalBytes - bytesWritten;
  if (remaining <= 0) return "Completed";
  const seconds = Math.ceil(remaining / speed);
  if (seconds < 60) return `${seconds}s remaining`;
  const minutes = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${minutes}m ${secs}s remaining`;
};

export default function FileCard({
  file,
  onToggleActions,
  onRemoveFile,
  onStartSharing,
  onStopSharing,
  onToggleShareInternet,
  onToggleShareNearby,
  onTogglePasswordProtected,
  onChangePasswordValue,
  onChangeNoteValue,
}: FileCardProps) {
  const [copiedLink, setCopiedLink] = useState<string | null>(null);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  useEffect(() => {
    if (file.isSharing) {
      setIsDropdownOpen(true);
    } else {
      setIsDropdownOpen(false);
    }
  }, [file.isSharing]);

  const handleCopyLinkText = async (linkText: string | null) => {
    if (!linkText) return;
    try {
      await navigator.clipboard.writeText(linkText);
      setCopiedLink(linkText);
      setTimeout(() => setCopiedLink(null), 1500);
    } catch (err) {
      console.error("Failed to copy link:", err);
    }
  };

  const handleDisconnect = async (sessionId: string) => {
    if (!file.shareLink && !file.localShareLink) return;
    const token = (file.shareLink || file.localShareLink || "").split("/").pop();
    if (!token) return;
    try {
      await disconnectDownloader(token, sessionId);
    } catch (err) {
      console.error("Failed to disconnect downloader:", err);
    }
  };

  // Determine state based on properties
  const isPreparing = file.shareCreating;
  const isUploading = file.isDownloading;
  const isSharing = file.isSharing && (!!file.shareLink || !!file.localShareLink);
  const isReady = !isSharing && !isPreparing;

  const activeDownloads = (file.activeDownloads && file.activeDownloads.length > 0)
    ? file.activeDownloads.map((dl, idx) => ({
      id: dl.session_id || `dl-${idx}`,
      name: dl.session_id || `Downloader #${idx + 1}`,
      bytesWritten: dl.bytes_written,
      totalBytes: dl.total_bytes,
      speed: dl.speed,
    }))
    : [];



  // Determine label and style class for metadata status
  let statusLabel = "";
  let statusClass = "";
  if (isPreparing) {
    statusLabel = "Preparing";
    statusClass = "preparing";
  } else if (isUploading) {
    statusLabel = "Uploading";
    statusClass = "uploading";
  } else if (isSharing) {
    statusLabel = "Sharing";
    statusClass = "sharing";
  }

  return (
    <div className={`file-card-container ${isSharing ? "is-sharing" : ""}`}>
      <div className="file-card-row">
        {/* Extension Badge */}
        <div className={`file-visual ${getExtClass(file.name)}`}>
          {getExtension(file.name) || "ZIP"}
        </div>

        {/* Meta Text */}
        <div className="file-meta-col">
          <h4 className="file-name" title={file.name}>{file.name}</h4>
          <span className="file-details">
            {file.size !== undefined ? formatFileSize(file.size) : "Size pending"}
            {statusLabel && (
              <>
                <span className="bullet-separator">•</span>
                <span className={`state-text ${statusClass}`}>{statusLabel}</span>
              </>
            )}
          </span>
        </div>

        {/* Center Progress/Status Area */}
        <div className="file-card-center-area">
        </div>

        {/* Action Buttons */}
        <div className="file-actions-col">
          {isReady ? (
            <>
              <button
                className="start-share-btn"
                onClick={onStartSharing}
                disabled={(!file.shareInternet && !file.shareNearby) || (file.passwordProtected && !file.passwordValue.trim())}
              >
                Start sharing
              </button>
              <button
                className={`settings-icon-only-btn ${file.isActionsOpen ? "active" : ""}`}
                onClick={onToggleActions}
                title="File Settings"
              >
                <SettingsIcon />
              </button>
              <button className="remove-x-btn" onClick={onRemoveFile} title="Remove">
                X
              </button>
            </>
          ) : isSharing ? (
            <>
              <button className="stop-sharing-red-btn" onClick={onStopSharing}>
                Stop sharing
              </button>
              <button
                className={`dropdown-toggle-btn ${isDropdownOpen ? "expanded" : ""}`}
                onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                title="Toggle details"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="6 9 12 15 18 9" />
                </svg>
              </button>
            </>
          ) : (
            // Fallback for preparing states
            !isPreparing && (
              <button className="close-action-btn" onClick={onRemoveFile} title="Remove transfer">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            )
          )}
        </div>
      </div>

      {file.shareError && (
        <p className="file-card-error" role="alert">
          {file.shareError}
        </p>
      )}

      {/* Settings Sub-panel */}
      <div className={`file-card-settings-wrapper ${file.isActionsOpen && !isSharing ? "open" : ""}`}>
        <div className="file-card-settings-panel">
          <div className="file-card-settings-inner">
            <div className="settings-panel-header">Sharing Options</div>
            <div className="settings-toggles">
              <label className="settings-toggle-row">
                <input
                  type="checkbox"
                  checked={file.shareInternet}
                  onChange={onToggleShareInternet}
                />
                <div className="toggle-label-meta">
                  <span className="toggle-label-title">Over the Internet</span>
                  <span className="toggle-label-desc">Generates a public Cloudflare tunnel link</span>
                </div>
              </label>
              <label className="settings-toggle-row">
                <input
                  type="checkbox"
                  checked={file.shareNearby}
                  onChange={onToggleShareNearby}
                />
                <div className="toggle-label-meta">
                  <span className="toggle-label-title">Nearby Sharing</span>
                  <span className="toggle-label-desc">Generates a local IP link for same router connections</span>
                </div>
              </label>

              <div className="settings-panel-divider" />

              <label className="settings-toggle-row">
                <input
                  type="checkbox"
                  checked={file.passwordProtected}
                  onChange={onTogglePasswordProtected}
                />
                <div className="toggle-label-meta">
                  <span className="toggle-label-title">Password Protection</span>
                  <span className="toggle-label-desc">Require visitors to enter a password to download</span>
                </div>
              </label>

              {file.passwordProtected && (
                <div className="settings-inputs-group">
                  <div className="settings-input-container">
                    <span className="settings-input-label">Password</span>
                    <div className="password-input-wrapper">
                      <input
                        type={showPassword ? "text" : "password"}
                        className="settings-text-input"
                        placeholder="Enter password"
                        value={file.passwordValue}
                        onChange={(e) => onChangePasswordValue(e.target.value)}
                      />
                      <button
                        type="button"
                        className="password-toggle-eye-btn"
                        onClick={() => setShowPassword(!showPassword)}
                      >
                        {showPassword ? (
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                            <line x1="1" y1="1" x2="23" y2="23" />
                          </svg>
                        ) : (
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                            <circle cx="12" cy="12" r="3" />
                          </svg>
                        )}
                      </button>
                    </div>
                  </div>
                  <div className="settings-input-container">
                    <span className="settings-input-label">Custom Note</span>
                    <textarea
                      className="settings-textarea-input"
                      placeholder="Add a download message or note"
                      value={file.noteValue}
                      onChange={(e) => onChangeNoteValue(e.target.value)}
                    />
                  </div>
                </div>
              )}
            </div>
            {!file.shareInternet && !file.shareNearby && (
              <div className="settings-warning-text">
                Please select at least one option to start sharing.
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Accordion details dropdown */}
      {isSharing && (
        <div className={`file-card-dropdown-wrapper ${isDropdownOpen ? "open" : ""}`}>
          <div className="file-card-dropdown-content">
            {/* Share Link Row (Internet) */}
            {file.shareInternet && file.shareLink && (
              <div className="dropdown-link-row">
                <span className="dropdown-label">Internet Link</span>
                <div className="dropdown-link-input-group">
                  <input
                    type="text"
                    readOnly
                    value={file.shareLink}
                    className="dropdown-link-input"
                    onClick={(e) => (e.target as HTMLInputElement).select()}
                  />
                  <button className="dropdown-copy-btn" onClick={() => handleCopyLinkText(file.shareLink)}>
                    {copiedLink === file.shareLink ? "Copied" : "Copy"}
                  </button>
                </div>
              </div>
            )}

            {/* Local Share Link Row (Nearby) */}
            {file.shareNearby && file.localShareLink && (
              <div className="dropdown-link-row">
                <span className="dropdown-label">Local Link (Same Wi-Fi)</span>
                <div className="dropdown-link-input-group">
                  <input
                    type="text"
                    readOnly
                    value={file.localShareLink}
                    className="dropdown-link-input"
                    onClick={(e) => (e.target as HTMLInputElement).select()}
                  />
                  <button className="dropdown-copy-btn" onClick={() => handleCopyLinkText(file.localShareLink)}>
                    {copiedLink === file.localShareLink ? "Copied" : "Copy"}
                  </button>
                </div>
              </div>
            )}

            {/* Downloads Section */}
            <div className="dropdown-downloads-section">
              <div className="downloads-header">
                <span className="downloads-title">Active Downloads</span>
                <span className="downloads-count-badge">
                  {activeDownloads.length} {activeDownloads.length === 1 ? "downloader" : "downloaders"}
                </span>
              </div>

              {activeDownloads.length > 0 ? (
                <div className="downloads-list">
                  {activeDownloads.map((dl, idx) => {
                    const dlPercent = dl.totalBytes > 0 ? (dl.bytesWritten / dl.totalBytes) * 100 : 0;
                    const neonClass = idx % 3 === 0 ? "neon-blue" : idx % 3 === 1 ? "neon-green" : "neon-purple";
                    const formattedSpeed = formatFileSize(dl.speed);
                    const formattedWritten = formatFileSize(dl.bytesWritten);
                    const formattedTotal = formatFileSize(dl.totalBytes);
                    const eta = getDownloaderEta(dl.bytesWritten, dl.totalBytes, dl.speed);

                    return (
                      <div key={dl.id} className="downloader-row">
                        <div className="downloader-info">
                          <span className="downloader-name">
                            {dl.name === "127.0.0.1" || dl.name === "::1" ? "Localhost Client" : `Downloader (${dl.name})`}
                          </span>
                          <span className="downloader-speed">
                            {formattedSpeed}/s
                          </span>
                        </div>
                        <div className="downloader-progress-container">
                          <div className="neon-progress-bar-container">
                            <div className={`neon-progress-fill ${neonClass}`} style={{ width: `${dlPercent}%` }} />
                          </div>
                          <div className="downloader-progress-details">
                            <span>{formattedWritten} of {formattedTotal} ({dlPercent.toFixed(0)}%)</span>
                            <span>{eta}</span>
                          </div>
                        </div>
                        <button
                          type="button"
                          className="disconnect-btn"
                          onClick={() => handleDisconnect(dl.id)}
                          style={{
                            marginTop: "8px",
                            alignSelf: "flex-start",
                            background: "none",
                            border: "none",
                            color: "#ef4444",
                            cursor: "pointer",
                            padding: "2px 6px",
                            fontSize: "11px",
                            fontWeight: 600,
                            borderRadius: "4px",
                            display: "inline-flex",
                            alignItems: "center",
                            transition: "background-color 0.15s ease"
                          }}
                        >
                          Disconnect
                        </button>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="no-downloads-placeholder">
                  No active downloads. Share the link above to start transferring files.
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
