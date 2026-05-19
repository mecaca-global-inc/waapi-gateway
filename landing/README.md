# waapi.link landing page

Single-file static site for **waapi.link**. Built with Tailwind via CDN — no build step.

## Local preview

```bash
cd landing
python3 -m http.server 8080
# open http://localhost:8080
```

## Deploy to Vercel

```bash
cd landing
vercel deploy --prod
# then in Vercel dashboard: assign domain waapi.link
```

Or wire it directly:

1. Vercel → New Project → Import the `waapi-gateway` repo.
2. **Root Directory:** `landing`
3. **Framework Preset:** Other / Static
4. Build & Output: leave empty (it's plain HTML).
5. Domains → add `waapi.link` and `www.waapi.link`.

## Stack

- Tailwind CSS v3 via CDN (`cdn.tailwindcss.com`)
- JetBrains Mono via Google Fonts
- Zero JS dependencies — just inline `tailwind.config` + CSS keyframes
- Shields.io badges for GitHub stars / forks / release / CI / license

## Editing

Everything is in `index.html`. To change colors edit the `tailwind.config` block at the top.
