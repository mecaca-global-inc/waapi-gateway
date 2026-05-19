"use client";

import { use, FormEvent, useState } from "react";
import useSWR from "swr";
import Shell from "@/components/Shell";
import Badge from "@/components/Badge";
import { QRCodeCanvas } from "qrcode.react";
import { api, fetcher } from "@/lib/api";
import type { Session } from "@/lib/types";

type QRResp = { status: string; code: string };

export default function SessionDetail({
  params,
}: {
  params: Promise<{ name: string }>;
}) {
  const { name } = use(params);
  const { data: sess, mutate } = useSWR<Session>(`/api/sessions/${name}`, fetcher, {
    refreshInterval: 2000,
  });
  const { data: qr } = useSWR<QRResp>(`/api/${name}/auth/qr`, fetcher, {
    refreshInterval: 2500,
  });

  return (
    <Shell>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold">{name}</h1>
        {sess && <Badge status={sess.status} />}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card title="Authentication">
          {qr?.code ? (
            <div className="flex flex-col items-center gap-3">
              <div className="bg-white p-4 rounded">
                <QRCodeCanvas value={qr.code} size={240} />
              </div>
              <p className="text-sm text-zinc-500">Scan with WhatsApp → Linked Devices.</p>
            </div>
          ) : sess?.status === "WORKING" ? (
            <p className="text-sm text-zinc-500">Session is connected as {sess.jid}.</p>
          ) : (
            <p className="text-sm text-zinc-500">No QR yet. Start the session first.</p>
          )}
          <PairCodeForm session={name} />
        </Card>

        <Card title="Send message">
          <SendForm session={name} onSent={() => mutate()} />
        </Card>
      </div>
    </Shell>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="border border-zinc-200 dark:border-zinc-800 rounded-lg p-5 bg-white dark:bg-zinc-900">
      <h2 className="font-medium mb-4">{title}</h2>
      {children}
    </div>
  );
}

function PairCodeForm({ session }: { session: string }) {
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setCode(null);
    try {
      const r = await api<{ code: string }>(`/api/${session}/auth/request-code`, {
        method: "POST",
        json: { phone },
      });
      setCode(r.code);
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "failed");
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-4 border-t border-zinc-200 dark:border-zinc-800 pt-4">
      <div className="text-sm font-medium mb-2">Pair by phone</div>
      <div className="flex gap-2">
        <input
          className="flex-1 px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          placeholder="6281234567890"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
        />
        <button className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 px-3 py-2 rounded">
          Get code
        </button>
      </div>
      {code && <div className="mt-2 text-lg font-mono">{code}</div>}
      {err && <div className="mt-2 text-sm text-red-600">{err}</div>}
    </form>
  );
}

function SendForm({ session, onSent }: { session: string; onSent: () => void }) {
  const [chat, setChat] = useState("");
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [last, setLast] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr(null);
    setLast(null);
    try {
      const r = await api<{ id: string }>("/api/sendText", {
        method: "POST",
        json: { session, chat_id: chat, text },
      });
      setLast(r.id);
      setText("");
      onSent();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "send failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-3">
      <input
        className="w-full px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
        placeholder="chat id (e.g. 112537404182586@lid, 628...@s.whatsapp.net, ...@g.us)"
        value={chat}
        onChange={(e) => setChat(e.target.value)}
      />
      <textarea
        className="w-full px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent h-24"
        placeholder="message"
        value={text}
        onChange={(e) => setText(e.target.value)}
      />
      <button
        disabled={busy}
        className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 px-4 py-2 rounded disabled:opacity-50"
      >
        {busy ? "Sending..." : "Send"}
      </button>
      {last && <div className="text-xs text-zinc-500">Sent id: {last}</div>}
      {err && <div className="text-sm text-red-600">{err}</div>}
    </form>
  );
}
