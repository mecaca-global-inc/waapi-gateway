"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { clearKey, getKey } from "@/lib/api";

const nav = [
  { href: "/sessions", label: "Sessions" },
  { href: "/webhooks", label: "Webhooks" },
  { href: "/docs", label: "API Docs" },
  { href: "/settings", label: "Settings" },
];

export default function Shell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  const router = useRouter();
  const [version, setVersion] = useState("");
  useEffect(() => {
    if (!getKey()) router.replace("/login");
  }, [router]);
  useEffect(() => {
    fetch("/healthz")
      .then((r) => r.json())
      .then((d) => setVersion(d.version ?? ""))
      .catch(() => {});
  }, []);

  return (
    <div className="flex min-h-screen">
      <aside className="w-56 border-r border-zinc-200 dark:border-zinc-800 px-4 py-6">
        <div className="text-lg font-semibold mb-6">WAAPI Gateway</div>
        <nav className="flex flex-col gap-1">
          {nav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`px-3 py-2 rounded text-sm ${
                path.startsWith(item.href)
                  ? "bg-zinc-200 dark:bg-zinc-800 font-medium"
                  : "hover:bg-zinc-100 dark:hover:bg-zinc-900"
              }`}
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <button
          onClick={() => {
            clearKey();
            router.replace("/login");
          }}
          className="mt-8 text-sm text-zinc-500 hover:underline"
        >
          Logout
        </button>
        {version && (
          <div
            className="mt-4 text-xs text-zinc-400 dark:text-zinc-600 font-mono"
            title={version}
          >
            {version.slice(0, 7)}
          </div>
        )}
      </aside>
      <main className="flex-1 p-8">{children}</main>
    </div>
  );
}
