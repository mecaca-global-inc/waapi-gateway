"use client";

import {
  KeyboardEvent,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { WEBHOOK_EVENTS, WebhookEvent, findEvent } from "@/lib/events";

type Props = {
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
};

export default function EventPicker({
  value,
  onChange,
  placeholder = "Type to filter events…",
}: Props) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const wrapRef = useRef<HTMLDivElement>(null);
  const listId = useId();

  const available = useMemo(() => {
    const picked = new Set(value);
    const q = query.trim().toLowerCase();
    return WEBHOOK_EVENTS.filter((e) => !picked.has(e.id)).filter(
      (e) =>
        q === "" ||
        e.id.includes(q) ||
        e.label.toLowerCase().includes(q) ||
        e.description.toLowerCase().includes(q),
    );
  }, [value, query]);

  useEffect(() => {
    if (highlight >= available.length) setHighlight(Math.max(0, available.length - 1));
  }, [available, highlight]);

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  function add(id: string) {
    if (value.includes(id)) return;
    onChange([...value, id]);
    setQuery("");
    setHighlight(0);
    inputRef.current?.focus();
  }

  function remove(id: string) {
    onChange(value.filter((v) => v !== id));
    inputRef.current?.focus();
  }

  function onKey(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setOpen(true);
      setHighlight((h) => Math.min(available.length - 1, h + 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setOpen(true);
      setHighlight((h) => Math.max(0, h - 1));
    } else if (e.key === "Enter") {
      if (open && available[highlight]) {
        e.preventDefault();
        add(available[highlight].id);
      }
    } else if (e.key === "Escape") {
      setOpen(false);
    } else if (e.key === "Backspace" && query === "" && value.length > 0) {
      e.preventDefault();
      remove(value[value.length - 1]);
    }
  }

  // group available items for nicer UX
  const grouped = useMemo(() => {
    const map = new Map<string, WebhookEvent[]>();
    available.forEach((e) => {
      const arr = map.get(e.group) ?? [];
      arr.push(e);
      map.set(e.group, arr);
    });
    return Array.from(map.entries());
  }, [available]);

  // flatten with group headers to align with highlight index
  const flat = useMemo(() => available, [available]);

  return (
    <div ref={wrapRef} className="relative">
      <div
        className="flex flex-wrap items-center gap-1.5 min-h-[42px] px-2 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 bg-transparent focus-within:ring-1 focus-within:ring-zinc-500 focus-within:border-zinc-500"
        onClick={() => {
          inputRef.current?.focus();
          setOpen(true);
        }}
        role="group"
        aria-label="Selected webhook events"
      >
        {value.map((id) => {
          const ev = findEvent(id);
          return (
            <span
              key={id}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-zinc-200 dark:bg-zinc-800 text-xs font-medium"
            >
              <span className="font-mono">{id}</span>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  remove(id);
                }}
                aria-label={`Remove ${ev?.label ?? id}`}
                className="rounded-full w-4 h-4 grid place-items-center text-zinc-500 hover:bg-zinc-300 dark:hover:bg-zinc-700 hover:text-zinc-900 dark:hover:text-zinc-100 transition"
              >
                ×
              </button>
            </span>
          );
        })}
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
            setHighlight(0);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKey}
          placeholder={value.length === 0 ? placeholder : ""}
          className="flex-1 min-w-[8ch] bg-transparent outline-none text-sm py-1"
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          aria-autocomplete="list"
          aria-activedescendant={
            open && flat[highlight] ? `${listId}-opt-${flat[highlight].id}` : undefined
          }
        />
      </div>

      {open && (
        <div
          id={listId}
          role="listbox"
          aria-label="Available webhook events"
          className="absolute z-20 mt-1 w-full max-h-72 overflow-auto rounded border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-lg"
        >
          {flat.length === 0 ? (
            <div className="px-3 py-4 text-sm text-zinc-500 text-center">
              {value.length === WEBHOOK_EVENTS.length
                ? "All events selected"
                : "No events match your search"}
            </div>
          ) : (
            grouped.map(([group, items]) => (
              <div key={group}>
                <div className="sticky top-0 px-3 py-1 text-[10px] uppercase tracking-wider text-zinc-500 bg-zinc-50 dark:bg-zinc-950/70 border-b border-zinc-200 dark:border-zinc-800">
                  {group}
                </div>
                {items.map((ev) => {
                  const idx = flat.findIndex((x) => x.id === ev.id);
                  const active = idx === highlight;
                  return (
                    <button
                      key={ev.id}
                      id={`${listId}-opt-${ev.id}`}
                      type="button"
                      role="option"
                      aria-selected={active}
                      onMouseEnter={() => setHighlight(idx)}
                      onClick={(e) => {
                        e.stopPropagation();
                        add(ev.id);
                      }}
                      className={`w-full text-left px-3 py-2 text-sm flex flex-col gap-0.5 transition ${
                        active
                          ? "bg-zinc-100 dark:bg-zinc-800"
                          : "hover:bg-zinc-50 dark:hover:bg-zinc-900"
                      }`}
                    >
                      <span className="font-mono text-xs">{ev.id}</span>
                      <span className="text-zinc-500 text-xs">{ev.description}</span>
                    </button>
                  );
                })}
              </div>
            ))
          )}
          <div className="px-3 py-2 border-t border-zinc-200 dark:border-zinc-800 text-[11px] text-zinc-500 flex justify-between">
            <span>↑/↓ navigate · Enter to add · ⌫ remove last</span>
            <span>{value.length}/{WEBHOOK_EVENTS.length} selected</span>
          </div>
        </div>
      )}
    </div>
  );
}
