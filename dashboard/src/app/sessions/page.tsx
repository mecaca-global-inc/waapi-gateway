"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import useSWR from "swr";
import Shell from "@/components/Shell";
import Badge from "@/components/Badge";
import { api, fetcher } from "@/lib/api";
import type { Session } from "@/lib/types";

export default function SessionsPage() {
  const { data, mutate, error } = useSWR<Session[]>("/api/sessions", fetcher, {
    refreshInterval: 3000,
  });
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    if (!name) return;
    setBusy(true);
    try {
      await api("/api/sessions", { method: "POST", json: { name } });
      setName("");
      mutate();
    } finally {
      setBusy(false);
    }
  }

  async function act(action: string, n: string) {
    await api(`/api/sessions/${n}/${action}`, { method: "POST" });
    mutate();
  }

  async function del(n: string) {
    if (!confirm(`Delete session "${n}"?`)) return;
    await api(`/api/sessions/${n}`, { method: "DELETE" });
    mutate();
  }

  return (
    <Shell>
      <h1 className="text-2xl font-semibold mb-6">Sessions</h1>

      <form onSubmit={create} className="flex gap-2 mb-6 max-w-md">
        <input
          className="flex-1 px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          placeholder="session name (e.g. default)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button
          disabled={busy}
          className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 px-4 py-2 rounded disabled:opacity-50"
        >
          Create
        </button>
      </form>

      {error && <div className="text-red-600 mb-4">{String(error.message)}</div>}

      <div className="overflow-x-auto rounded border border-zinc-200 dark:border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-100 dark:bg-zinc-900 text-left">
            <tr>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">JID</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).map((s) => (
              <tr key={s.name} className="border-t border-zinc-200 dark:border-zinc-800">
                <td className="px-3 py-2 font-medium">
                  <Link href={`/sessions/${s.name}`} className="hover:underline">
                    {s.name}
                  </Link>
                </td>
                <td className="px-3 py-2 text-zinc-500">{s.jid || "—"}</td>
                <td className="px-3 py-2"><Badge status={s.status} /></td>
                <td className="px-3 py-2 text-right space-x-2">
                  <button onClick={() => act("start", s.name)} className="text-xs px-2 py-1 rounded bg-zinc-200 dark:bg-zinc-800">Start</button>
                  <button onClick={() => act("stop", s.name)} className="text-xs px-2 py-1 rounded bg-zinc-200 dark:bg-zinc-800">Stop</button>
                  <button onClick={() => act("logout", s.name)} className="text-xs px-2 py-1 rounded bg-zinc-200 dark:bg-zinc-800">Logout</button>
                  <button onClick={() => del(s.name)} className="text-xs px-2 py-1 rounded bg-red-200 dark:bg-red-900 text-red-900 dark:text-red-100">Delete</button>
                </td>
              </tr>
            ))}
            {(data ?? []).length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-zinc-500">No sessions yet.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
