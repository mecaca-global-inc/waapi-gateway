"use client";

import { FormEvent, useState } from "react";
import useSWR from "swr";
import Shell from "@/components/Shell";
import { api, fetcher } from "@/lib/api";
import type { Session, Webhook } from "@/lib/types";

export default function WebhooksPage() {
  const { data: sessions } = useSWR<Session[]>("/api/sessions", fetcher);
  const [picked, setPicked] = useState<string>("");
  const session = picked || sessions?.[0]?.name || "";
  const { data: hooks, mutate } = useSWR<Webhook[]>(
    session ? `/api/${session}/webhooks` : null,
    fetcher,
  );

  const [url, setUrl] = useState("");
  const [secret, setSecret] = useState("");
  const [events, setEvents] = useState("message,message.ack,session.status");

  async function add(e: FormEvent) {
    e.preventDefault();
    if (!session || !url) return;
    await api(`/api/${session}/webhooks`, {
      method: "POST",
      json: {
        url,
        secret,
        events: events.split(",").map((s) => s.trim()).filter(Boolean),
      },
    });
    setUrl("");
    setSecret("");
    mutate();
  }

  async function del(id: number) {
    await api(`/api/webhooks/${id}`, { method: "DELETE" });
    mutate();
  }

  return (
    <Shell>
      <h1 className="text-2xl font-semibold mb-6">Webhooks</h1>

      <div className="mb-4">
        <label className="text-sm mr-2">Session:</label>
        <select
          value={session}
          onChange={(e) => setPicked(e.target.value)}
          className="px-2 py-1 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
        >
          {(sessions ?? []).map((s) => (
            <option key={s.name} value={s.name}>
              {s.name}
            </option>
          ))}
        </select>
      </div>

      <form
        onSubmit={add}
        className="mb-6 grid grid-cols-1 md:grid-cols-4 gap-2 items-end"
      >
        <input
          className="px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent md:col-span-2"
          placeholder="https://your.endpoint/hook"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <input
          className="px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          placeholder="hmac secret (optional)"
          value={secret}
          onChange={(e) => setSecret(e.target.value)}
        />
        <input
          className="px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent md:col-span-3"
          placeholder="events (comma-separated, blank = all)"
          value={events}
          onChange={(e) => setEvents(e.target.value)}
        />
        <button className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 px-4 py-2 rounded">
          Add
        </button>
      </form>

      <div className="overflow-x-auto rounded border border-zinc-200 dark:border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-100 dark:bg-zinc-900 text-left">
            <tr>
              <th className="px-3 py-2">URL</th>
              <th className="px-3 py-2">Events</th>
              <th className="px-3 py-2">Enabled</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {(hooks ?? []).map((h) => (
              <tr key={h.id} className="border-t border-zinc-200 dark:border-zinc-800">
                <td className="px-3 py-2 break-all">{h.url}</td>
                <td className="px-3 py-2 text-zinc-500">{(h.events ?? []).join(", ") || "(all)"}</td>
                <td className="px-3 py-2">{h.enabled ? "yes" : "no"}</td>
                <td className="px-3 py-2 text-right">
                  <button
                    onClick={() => del(h.id)}
                    className="text-xs px-2 py-1 rounded bg-red-200 dark:bg-red-900 text-red-900 dark:text-red-100"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
            {(hooks ?? []).length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-zinc-500">
                  No webhooks for this session.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
