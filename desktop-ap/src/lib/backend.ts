const defaultBackend = "http://127.0.0.1:8787";

export function getBackendBaseUrl(): string {
  return import.meta.env.VITE_BACKEND_URL ?? defaultBackend;
}

async function parseError(res: Response): Promise<string> {
  try {
    const j = (await res.json()) as { error?: string };
    if (j.error) return j.error;
  } catch {
    /* ignore */
  }
  return res.statusText || `HTTP ${res.status}`;
}

export async function fetchBackendHealth(): Promise<{ status: string; tunnel_active: boolean }> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/health`);
  if (!res.ok) {
    throw new Error(`backend returned ${res.status}`);
  }
  return res.json() as Promise<{ status: string; tunnel_active: boolean }>;
}

export type NewRecipient = { display_name: string; email?: string };

export type CreateShareRequest = {
  paths: string[];
  recipient_person_ids?: number[];
  new_recipients?: NewRecipient[];
  label?: string;
  expires_at?: string;
  password?: string;
  note?: string;
  isInternet?: boolean;
  isLAN?: boolean;
};

export type CreateShareResponse = {
  token: string;
  download_url: string;
  local_download_url: string;
  share_id: number;
  public_base_url: string;
  password?: string;
  note?: string;
};

export async function createShare(body: CreateShareRequest): Promise<CreateShareResponse> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/v1/shares`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(await parseError(res));
  }
  const item = await res.json();
  return {
    token: item.token !== undefined ? item.token : item.Token,
    download_url: item.download_url !== undefined ? item.download_url : item.PublicDownloadURL,
    local_download_url: item.local_download_url !== undefined ? item.local_download_url : item.LocalDownloadURL,
    share_id: item.share_id !== undefined ? item.share_id : item.ShareID,
    public_base_url: item.public_base_url !== undefined ? item.public_base_url : item.PublicBaseURL,
    password: item.password !== undefined ? item.password : item.Password,
    note: item.note !== undefined ? item.note : item.Note,
  };
}

export type ShareListItem = {
  id: number;
  token: string;
  created_at: string;
  label?: string | null;
  download_url: string;
  local_download_url: string;
  file_count: number;
  total_size: number;
  primary_name: string;
  recipient_summary?: string | null;
  password?: string;
  note?: string;
};

export async function listShares(): Promise<{ shares: ShareListItem[]; tunnelActive: boolean }> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/v1/shares`);
  if (!res.ok) {
    throw new Error(await parseError(res));
  }
  const data = await res.json();
  const sharesList = data.shares || [];
  const mapped = sharesList.map((item: any) => ({
    id: item.id !== undefined ? item.id : item.ID,
    token: item.token !== undefined ? item.token : item.Token,
    created_at: item.created_at !== undefined ? item.created_at : item.CreatedAt,
    label: item.label !== undefined ? item.label : item.Label,
    download_url: item.download_url !== undefined ? item.download_url : item.PublicDownloadURL,
    local_download_url: item.local_download_url !== undefined ? item.local_download_url : item.LocalDownloadURL,
    file_count: item.file_count !== undefined ? item.file_count : item.FileCount,
    total_size: item.total_size !== undefined ? item.total_size : item.TotalSize,
    primary_name: item.primary_name !== undefined ? item.primary_name : item.PrimaryName,
    recipient_summary: item.recipient_summary !== undefined ? item.recipient_summary : item.RecipientSummary,
    password: item.password !== undefined ? item.password : item.Password,
    note: item.note !== undefined ? item.note : item.Note,
  }));
  return {
    shares: mapped,
    tunnelActive: data.tunnel_active ?? false,
  };
}

export async function deleteShare(token: string): Promise<void> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/v1/shares?token=${token}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    throw new Error(await parseError(res));
  }
}

export type TransferStats = {
  token: string;
  bytes_written: number;
  total_bytes: number;
  speed: number;
  is_active: boolean;
  session_id: string;
};

export type ActiveTransfersResponse = {
  [token: string]: TransferStats[];
};

export async function fetchTransfers(): Promise<ActiveTransfersResponse> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/v1/transfers`);
  if (!res.ok) {
    throw new Error(await parseError(res));
  }
  return res.json() as Promise<ActiveTransfersResponse>;
}

export async function disconnectDownloader(token: string, ip: string): Promise<void> {
  const base = getBackendBaseUrl().replace(/\/$/, "");
  const res = await fetch(`${base}/v1/transfers?token=${token}&ip=${ip}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    throw new Error(await parseError(res));
  }
}
