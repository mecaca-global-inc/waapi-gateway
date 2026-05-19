"use client";

import { FormEvent, useState } from "react";
import useSWR from "swr";
import Shell from "@/components/Shell";
import { api, fetcher } from "@/lib/api";
import type { APIKey } from "@/lib/types";

export default function SettingsPage() {
  const { data, mutate } = useSWR<APIKey[]>("/api/keys", fetcher);
  const [name, setName] = useState("");
  const [issued, setIssued] = useState<string | null>(null);

  async function create(e: FormEvent) {
    e.preventDefault();
    if (!name) return;
    const r = await api<{ id: number; name: string; api_key: string }>("/api/keys", {
      method: "POST",
      json: { name },
    });
    setIssued(r.api_key);
    setName("");
    mutate();
  }

  async function del(id: number) {
    if (!confirm("Delete key?")) return;
    await api(`/api/keys/${id}`, { method: "DELETE" });
    mutate();
  }

  return (
    <Shell>
      <h1 className="text-2xl font-semibold mb-6">API keys</h1>

      <form onSubmit={create} className="flex gap-2 mb-6 max-w-md">
        <input
          className="flex-1 px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          placeholder="key label"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <button className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 px-4 py-2 rounded">
          Create
        </button>
      </form>

      {issued && (
        <div className="mb-6 p-3 rounded border border-yellow-400 bg-yellow-50 dark:bg-yellow-950 text-sm">
          New key (copy now, will not be shown again):
          <code className="ml-2 font-mono break-all">{issued}</code>
        </div>
      )}

      <div className="overflow-x-auto rounded border border-zinc-200 dark:border-zinc-800">
        <table className="w-full text-sm">
          <thead className="bg-zinc-100 dark:bg-zinc-900 text-left">
            <tr>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">Created</th>
              <th className="px-3 py-2">Last used</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).map((k) => (
              <tr key={k.id} className="border-t border-zinc-200 dark:border-zinc-800">
                <td className="px-3 py-2 font-medium">{k.name}</td>
                <td className="px-3 py-2 text-zinc-500">{new Date(k.created_at * 1000).toLocaleString()}</td>
                <td className="px-3 py-2 text-zinc-500">{k.last_used ? new Date(k.last_used * 1000).toLocaleString() : "—"}</td>
                <td className="px-3 py-2 text-right">
                  <button
                    onClick={() => del(k.id)}
                    className="text-xs px-2 py-1 rounded bg-red-200 dark:bg-red-900 text-red-900 dark:text-red-100"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
            {(data ?? []).length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-zinc-500">
                  No API keys.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}
