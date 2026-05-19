"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setU] = useState("admin");
  const [password, setP] = useState("admin");
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    setBusy(true);
    try {
      await login(username, password);
      router.replace("/sessions");
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : "login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center">
      <form
        onSubmit={onSubmit}
        className="w-80 space-y-4 p-6 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900"
      >
        <h1 className="text-xl font-semibold">Sign in</h1>
        <input
          className="w-full px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          value={username}
          onChange={(e) => setU(e.target.value)}
          placeholder="username"
          autoComplete="username"
        />
        <input
          className="w-full px-3 py-2 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent"
          value={password}
          onChange={(e) => setP(e.target.value)}
          placeholder="password"
          type="password"
          autoComplete="current-password"
        />
        {err && <div className="text-sm text-red-600">{err}</div>}
        <button
          disabled={busy}
          className="w-full bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 py-2 rounded disabled:opacity-50"
        >
          {busy ? "Signing in..." : "Sign in"}
        </button>
      </form>
    </div>
  );
}
