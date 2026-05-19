export const GATEWAY_URL =
  process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:3000";

const KEY_STORAGE = "wa_gateway_key";

export function getKey(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(KEY_STORAGE);
}

export function setKey(k: string) {
  window.localStorage.setItem(KEY_STORAGE, k);
}

export function clearKey() {
  window.localStorage.removeItem(KEY_STORAGE);
}

type FetchOpts = RequestInit & { json?: unknown };

export async function api<T>(path: string, opts: FetchOpts = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(opts.headers as Record<string, string> | undefined),
  };
  const key = getKey();
  if (key) headers["Authorization"] = `Bearer ${key}`;

  let body = opts.body;
  if (opts.json !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.json);
  }
  const res = await fetch(`${GATEWAY_URL}${path}`, {
    ...opts,
    headers,
    body,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) return undefined as T;
  const ct = res.headers.get("content-type") ?? "";
  if (ct.includes("application/json")) return (await res.json()) as T;
  return (await res.text()) as unknown as T;
}

export const fetcher = <T,>(path: string) => api<T>(path);

export async function login(username: string, password: string) {
  const r = await fetch(`${GATEWAY_URL}/api/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!r.ok) throw new Error(await r.text());
  const data = (await r.json()) as { api_key: string };
  setKey(data.api_key);
  return data.api_key;
}
