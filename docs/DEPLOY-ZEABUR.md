# Deploy to Zeabur

Single-binary deploy (gateway + embedded dashboard). ~5 min total.

## Prerequisites

- GitHub account with the repo pushed
- [Zeabur](https://zeabur.com) account (sign in with GitHub)
- A strong `ADMIN_PASS` (not `admin` / `changeme` / `password` / empty)

## Step 1 — Install the Zeabur GitHub App

Open <https://github.com/apps/zeabur/installations/new>. Pick your account/org → grant access to `waapi-gateway` → Install.

## Step 2 — Create project + service

1. Zeabur dashboard → **Create Project** → pick region (Singapore for SEA)
2. **+ Add Service** → **GitHub** → select `waapi-gateway` → branch `main`
3. Zeabur auto-detects the root `Dockerfile`

## Step 3 — Environment variables

Service → **Variables** tab → paste:

```env
ADMIN_USER=admin
ADMIN_PASS=YourStrongPassword!23
CORS_ORIGINS=*
LOG_LEVEL=info
LOG_FORMAT=json
DEVICE_NAME=WAAPI Gateway
DB_DIALECT=sqlite3
DB_URI=file:/app/storages/gateway.db?_foreign_keys=on
```

Replace `ADMIN_PASS` with a real secret. **Don't set `PORT`** — Zeabur injects it, the binary reads it automatically.

## Step 4 — Persistent volume (critical)

Service → **Volumes** tab → **Add Volume**:

| Field           | Value             |
| --------------- | ----------------- |
| Volume ID       | `storages`        |
| Mount Directory | `/app/storages`   |

Without this, WhatsApp pairings reset on every redeploy.

## Step 5 — Public domain

Service → **Networking** → **Generate Domain**:

- Subdomain: `waapi` (or anything available)
- Port: **8080** (Zeabur's injected `$PORT`)
- Click **Confirm**

Gives you `https://<subdomain>.zeabur.app`. HTTPS auto-provisioned.

## Step 6 — Deploy

Zeabur builds + boots automatically on first save (and every `git push` after). Watch the **Logs** tab. Expect:

```
INF starting waapi-gateway addr=:3000 db=sqlite3 device="WAAPI Gateway"
INF http listen listen=:8080
```

## Step 7 — Verify

```bash
curl https://<your-subdomain>.zeabur.app/healthz
# → {"ok":true}
```

Open `https://<your-subdomain>.zeabur.app` in a browser → login with `admin` / your `ADMIN_PASS` → dashboard renders.

## Step 8 — First WhatsApp pairing

1. Dashboard → **Sessions** → enter name e.g. `main` → **Create**
2. Click the `main` row → **Start**
3. QR appears → scan with phone (WhatsApp → *Linked Devices* → *Link a Device*)
4. Status flips `SCAN_QR` → `WORKING`

## Step 9 — Push-to-deploy

Already wired. Any `git push origin main` triggers a Zeabur rebuild. No further setup.

## Optional — Custom domain

Service → **Networking** → **Add Domain** → enter `api.your-domain.com` → Zeabur shows the DNS record → add it at your registrar → HTTPS auto-issued in ~2 min.

After the custom domain is live, also tighten CORS:

```env
CORS_ORIGINS=https://api.your-domain.com,https://your-domain.com
```

## Plan / cost

| Tier        | Use case                          | Cost              |
| ----------- | --------------------------------- | ----------------- |
| Free/Hobby  | dev, sleeps after inactivity      | $0                |
| Developer   | always-on (needed for WhatsApp WS)| ~$5/mo per service|
| + Volume    | 1 GB SQLite + media               | included          |

**Always-on plan required for production** — WhatsApp drops the session if the process sleeps.

## Troubleshooting

| Symptom                              | Fix                                                                  |
| ------------------------------------ | -------------------------------------------------------------------- |
| 502 from Cloudflare                  | Check **Logs**. Usually weak `ADMIN_PASS` aborting boot.             |
| QR re-scan demanded after redeploy   | Volume not mounted; redo Step 4.                                     |
| Dashboard 404 on sub-pages           | Embedded build missing; redeploy from latest `main`.                 |
| `invalid api key` on dashboard       | localStorage stale — click **Logout** then login again.              |
| `ADMIN_PASS is weak` in logs         | Replace with a real password (not on the blocklist), save, redeploy. |
| Container restarting in a loop       | Inspect **Logs** for the first error line — usually env or volume.   |

## What gets deployed

- **One container** running `waapi-gateway`
- Listens on `$PORT` (Zeabur's `8080`)
- Same origin serves:
  - `/`           — dashboard (Next.js static export, embedded in the binary)
  - `/api/*`      — REST API
  - `/ws`         — WebSocket event stream
  - `/openapi.yaml` — OpenAPI 3.1 spec
  - `/healthz`    — liveness probe
- SQLite at `/app/storages/gateway.db` (persists via the volume)
- WhatsApp credentials persist in the same `/app/storages` volume

## Updating

```bash
# locally
git add -A
git commit -m "feat: ..."
git push origin main
```

Zeabur picks up the push, rebuilds, and deploys. Healthcheck protects against bad builds — if `/healthz` doesn't return 200, traffic continues hitting the previous container.
