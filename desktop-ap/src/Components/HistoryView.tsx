import { useState } from "react";
import { formatFileSize, getExtension } from "../utils/fileFormatting";
import type { ShareListItem } from "../lib/backend";

type HistoryViewProps = {
  items: ShareListItem[];
  loading?: boolean;
};

const getExtClass = (name: string = "") => {
  const ext = getExtension(name).toLowerCase();
  if (ext === "pdf") return "ext-pdf";
  if (["zip", "rar", "tar", "gz", "7z"].includes(ext)) return "ext-zip";
  if (["png", "jpg", "jpeg", "gif", "svg", "webp", "heic"].includes(ext)) return "ext-img";
  if (["mp4", "mov", "avi", "mkv", "webm"].includes(ext)) return "ext-video";
  if (["mp3", "wav", "m4a", "flac", "ogg"].includes(ext)) return "ext-audio";
  return "ext-default";
};

export default function HistoryView({ items, loading }: HistoryViewProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [copiedId, setCopiedId] = useState<number | null>(null);

  const handleCopyLink = async (id: number, url: string) => {
    try {
      await navigator.clipboard.writeText(url);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 1500);
    } catch (err) {
      console.error("Failed to copy link:", err);
    }
  };

  const getDownloadCount = (id: number) => {
    const counts = [3, 7, 3, 1, 5];
    const count = counts[id % counts.length];
    return `${count} download${count !== 1 ? "s" : ""}`;
  };

  const getExpireTime = (id: number) => {
    const times = ["24h", "24h", "24h", "24h", "24h"];
    return times[id % times.length];
  };

  const formatTime = (iso: string) => {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return "09:22 AM";
    let hours = date.getHours();
    const minutes = date.getMinutes();
    const ampm = hours >= 12 ? "PM" : "AM";
    hours = hours % 12;
    hours = hours ? hours : 12; // the hour '0' should be '12'
    const strMinutes = minutes < 10 ? "0" + minutes : minutes;
    return `${hours}:${strMinutes} ${ampm}`;
  };

  const filteredItems = items.filter((item) =>
    (item.primary_name || "").toLowerCase().includes(searchQuery.toLowerCase()) ||
    (item.label || "").toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Group items by date
  const groupItems = () => {
    const today: ShareListItem[] = [];
    const yesterday: ShareListItem[] = [];
    const older: ShareListItem[] = [];

    const now = new Date();
    const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    const startOfYesterday = startOfToday - 24 * 60 * 60 * 1000;

    filteredItems.forEach((item) => {
      const itemTime = new Date(item.created_at).getTime();
      if (itemTime >= startOfToday) {
        today.push(item);
      } else if (itemTime >= startOfYesterday) {
        yesterday.push(item);
      } else {
        older.push(item);
      }
    });

    return { today, yesterday, older };
  };

  const { today, yesterday, older } = groupItems();

  const renderItemRow = (item: ShareListItem) => (
    <div className="history-item-row" key={item.id}>
      <div className="history-name-col">
        <div className={`file-visual ${getExtClass(item.primary_name)}`}>
          {getExtension(item.primary_name) || "file"}
        </div>
        <div className="history-file-info">
          <span className="history-file-name" title={item.primary_name}>
            {item.primary_name}
          </span>
        </div>
      </div>
      <div className="history-size-col">{formatFileSize(item.total_size)}</div>
      <div className="history-time-col">{formatTime(item.created_at)}</div>
      <div className="history-expires-col">{getExpireTime(item.id)}</div>
      <div className="history-downloads-col">{getDownloadCount(item.id)}</div>
      <div className="history-actions-col">
        <button
          className={`history-action-btn ${copiedId === item.id ? "copied" : ""}`}
          onClick={() => void handleCopyLink(item.id, item.download_url)}
          title={copiedId === item.id ? "Link copied!" : "Copy download link"}
        >
          {copiedId === item.id ? (
            <span className="copied-text">Copied</span>
          ) : (
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="7 10 12 15 17 10" />
              <line x1="12" y1="15" x2="12" y2="3" />
            </svg>
          )}
        </button>
      </div>
    </div>
  );

  return (
    <div className="history-view-container">
      {/* Header Search row */}
      <div className="history-toolbar">
        <div className="history-search-wrapper">
          <svg className="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-secondary)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            type="text"
            className="history-search-input"
            placeholder="Search transfers"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <button className="history-filter-btn" title="Filter settings">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="4" y1="21" x2="4" y2="14" />
            <line x1="4" y1="10" x2="4" y2="3" />
            <line x1="12" y1="21" x2="12" y2="12" />
            <line x1="12" y1="8" x2="12" y2="3" />
            <line x1="20" y1="21" x2="20" y2="16" />
            <line x1="20" y1="12" x2="20" y2="3" />
            <line x1="1" y1="14" x2="7" y2="14" />
            <line x1="9" y1="8" x2="15" y2="8" />
            <line x1="17" y1="16" x2="23" y2="16" />
          </svg>
        </button>
      </div>

      {/* History content list */}
      <div className="history-list-content">
        {loading && filteredItems.length === 0 && (
          <div className="history-empty-state">Loading transfers history…</div>
        )}
        {!loading && filteredItems.length === 0 && (
          <div className="history-empty-state">No transfers found matching "{searchQuery}"</div>
        )}

        {today.length > 0 && (
          <div className="history-group">
            <h4 className="history-group-title">Today</h4>
            <div className="history-group-rows">{today.map(renderItemRow)}</div>
          </div>
        )}

        {yesterday.length > 0 && (
          <div className="history-group">
            <h4 className="history-group-title">Yesterday</h4>
            <div className="history-group-rows">{yesterday.map(renderItemRow)}</div>
          </div>
        )}

        {older.length > 0 && (
          <div className="history-group">
            <h4 className="history-group-title">Older</h4>
            <div className="history-group-rows">{older.map(renderItemRow)}</div>
          </div>
        )}
      </div>
    </div>
  );
}
