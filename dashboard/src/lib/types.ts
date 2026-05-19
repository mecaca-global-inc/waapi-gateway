export type Session = {
  name: string;
  jid: string;
  status: "STOPPED" | "STARTING" | "SCAN_QR" | "WORKING" | "FAILED";
};

export type Webhook = {
  id: number;
  session_name: string;
  url: string;
  secret: string;
  events: string[] | null;
  enabled: boolean;
  created_at: number;
};

export type APIKey = {
  id: number;
  name: string;
  created_at: number;
  last_used?: number;
};
