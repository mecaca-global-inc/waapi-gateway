"use client";

import { useEffect, useRef } from "react";
import Shell from "@/components/Shell";
import { GATEWAY_URL, getKey } from "@/lib/api";

declare global {
  interface Window {
    SwaggerUIBundle?: (opts: Record<string, unknown>) => unknown;
  }
}

export default function DocsPage() {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cssLink: HTMLLinkElement | null = null;
    let script: HTMLScriptElement | null = null;

    function render() {
      if (!ref.current || !window.SwaggerUIBundle) return;
      const key = getKey();
      window.SwaggerUIBundle({
        url: `${GATEWAY_URL}/openapi.yaml`,
        domNode: ref.current,
        deepLinking: true,
        persistAuthorization: true,
        tryItOutEnabled: true,
        requestInterceptor: (req: { headers: Record<string, string> }) => {
          if (key) req.headers["Authorization"] = `Bearer ${key}`;
          return req;
        },
      });
    }

    if (window.SwaggerUIBundle) {
      render();
    } else {
      cssLink = document.createElement("link");
      cssLink.rel = "stylesheet";
      cssLink.href = "https://unpkg.com/swagger-ui-dist@5/swagger-ui.css";
      document.head.appendChild(cssLink);

      script = document.createElement("script");
      script.src = "https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js";
      script.async = true;
      script.onload = render;
      document.body.appendChild(script);
    }
  }, []);

  return (
    <Shell>
      <h1 className="text-2xl font-semibold mb-4">API Docs</h1>
      <p className="text-sm text-zinc-500 mb-4">
        Your dashboard API key is auto-injected into every Try-it-out request.
      </p>
      <div ref={ref} className="bg-white rounded border border-zinc-200 dark:border-zinc-800 p-2" />
    </Shell>
  );
}
