# WAAPI Gateway

[![CI](https://github.com/techsupport-mecaca/waapi-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/techsupport-mecaca/waapi-gateway/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.26%2B-00ADD8)](https://go.dev/)
[![Node](https://img.shields.io/badge/node-22%2B-339933)](https://nodejs.org/)

Self-hosted **WhatsApp HTTP API + Dashboard**, powered by [tulir/whatsmeow](https://github.com/tulir/whatsmeow). A leaner alternative to [WAHA](https://github.com/devlikeapro/waha) / [GOWA](https://github.com/aldinokemal/go-whatsapp-web-multidevice) — single Go binary plus a Next.js dashboard.

> ⚠️ **Unofficial project.** Not affiliated with WhatsApp / Meta. For commercial workloads consider the official WhatsApp Business API. Use of this software is your responsibility.

## Features

- 🔌 **Multi-session** — run many WhatsApp accounts per gateway instance
- 📱 **QR + pairing-code login**
- 💬 **Send** text, image, video, voice (PTT), file, location, contact
- 👀 **Receipts & presence** — mark-as-read, typing indicators
- ⏳ **Disappearing-messages aware** — auto-inherits the chat's ephemeral timer so replies don't trigger the "sender may be on an old version" warning
- 📥 **Webhook delivery** with HMAC-SHA256 signing, retries, event filtering
- 🔴 **Live WebSocket stream** for the dashboard
- 🎨 **Next.js 16 dashboard** — sessions, send playground, webhooks, API keys, embedded Swagger UI
- 🗄️ **SQLite default**, Postgres via `DB_URI`
- 🐳 **Docker / Docker Compose** ready

## Quick start (Docker)

```bash
git clone https://github.com/techsupport-mecaca/waapi-gateway
cd waapi-gateway
cp .env.example .env
# Edit ADMIN_PASS in .env — gateway refuses to boot with weak defaults
docker compose up -d --build
```

- Dashboard: <http://localhost:3001>
- Gateway API: <http://localhost:3000>
- Swagger UI: <http://localhost:3001/docs> (auth-gated)

## Local development

```bash
go version    # 1.26+
node -v       # 22+

cp .env.example .env
# set a strong ADMIN_PASS

# terminal 1
go run ./cmd/server

# terminal 2
cd dashboard
npm install
npm run dev -- -p 3001
```

## Authentication

Two layers:

| Layer | Audience | Method |
|---|---|---|
| `POST /api/login` | Dashboard / first-time CLI users | Username + password (`ADMIN_USER`/`ADMIN_PASS`) → returns an API key |
| `Authorization: Bearer <key>` *or* `X-Api-Key: <key>` | All `/api/*` calls | API key (managed in dashboard *Settings*) |

WebSocket `/ws` accepts the key via query param: `/ws?key=<api_key>`.

## REST API

Full interactive reference: **`/docs` in the dashboard** (loads from `/openapi.yaml`).

Highlights:

| Category | Endpoint |
|---|---|
| Sessions | `GET/POST /api/sessions`, `POST /api/sessions/{name}/{start\|stop\|logout}` |
| Login    | `GET /api/{session}/auth/qr`, `GET /api/{session}/auth/qr.png`, `POST /api/{session}/auth/request-code` |
| Send     | `POST /api/sendText\|sendImage\|sendVideo\|sendVoice\|sendFile\|sendLocation\|sendContact` |
| Read     | `GET /api/{session}/{me\|contacts\|chats\|groups}` |
| Webhooks | `GET/POST /api/{session}/webhooks`, `DELETE /api/webhooks/{id}` |
| Keys     | `GET/POST /api/keys`, `DELETE /api/keys/{id}` |
| Stream   | `GET /ws?key=...` |
| Health   | `GET /healthz`, `GET /readyz` |

`chat_id` accepts:
- bare phone (`6281234567890`) → auto-suffixed `@s.whatsapp.net`
- full JID: `6281234567890@s.whatsapp.net`, `112537404182586@lid`, `1234567890-1700000000@g.us`

## Webhook payload

```json
{
  "event": "message",
  "session": "default",
  "timestamp": 1747100000,
  "payload": {
    "id": "3EB0...",
    "chat": "6281234567890@s.whatsapp.net",
    "sender": "6281234567890@s.whatsapp.net",
    "from_me": false,
    "timestamp": 1747100000,
    "push_name": "Jane",
    "body": "hi",
    "has_media": false
  }
}
```

Each request carries:

```
X-Webhook-Signature: sha256=<hex_hmac_sha256(secret, raw_body)>
```

Verify it on your side before trusting the payload.

## Configuration (env)

| Var | Default | Notes |
|---|---|---|
| `HTTP_ADDR` | `:3000` | listen address |
| `DB_DIALECT` | `sqlite3` | or `postgres` |
| `DB_URI` | `file:storages/gateway.db?_foreign_keys=on` | sqlite path or postgres URL |
| `ADMIN_USER` | `admin` | dashboard login |
| `ADMIN_PASS` | *(required, no weak defaults)* | refuses `admin`/`changeme`/`password`/empty |
| `ALLOW_WEAK_AUTH` | *(unset)* | set to `1` to bypass weak-password guard (dev only) |
| `CORS_ORIGINS` | `*` | comma-separated; set explicit origins in prod |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `console` | `json` for structured prod logs |
| `MEDIA_DIR` | `storages/media` | temp media work dir |
| `NEXT_PUBLIC_GATEWAY_URL` | `http://localhost:3000` | build-time, dashboard only |

## Project layout

```
cmd/server/         Go entry point
internal/config/    env loader + weak-password guard
internal/store/     SQLite/Postgres + goose migrations + repo
internal/wa/        whatsmeow wrappers (manager, session, events)
internal/webhook/   HMAC-signed outbound delivery + retry
internal/api/       Fiber routes (REST, WS, OpenAPI spec, login rate-limit)
dashboard/          Next.js 16 + Tailwind v4 + SWR (incl. /docs Swagger UI)
.github/workflows/  CI (Go build/test, dashboard build, multi-arch Docker)
```

## n8n / Zapier integration

1. In dashboard → **Webhooks** → add your n8n Webhook URL, optional `secret`, events `message`.
2. n8n flow:
   - **Webhook** trigger (POST)
   - **IF** node: `{{ $json.event === "message" && !$json.payload.from_me }}` (prevents reply loops)
   - **HTTP Request**: `POST http://host.docker.internal:3000/api/sendText`
     - Header: `Authorization: Bearer <api_key>`
     - JSON body:
       ```json
       {
         "session": "{{ $json.session }}",
         "chat_id": "{{ $json.payload.chat }}",
         "text": "Echo: {{ $json.payload.body }}"
       }
       ```

## Security

See [SECURITY.md](SECURITY.md). Report vulnerabilities to **security@waapi.link** — please don't open public issues.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE).
