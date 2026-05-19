# Deploy WAAPI Gateway on Zeabur

Self-host in under 5 minutes. No clone, no build, no Dockerfile tweaks.

## TL;DR — One-click install

1. **Sign in** at [zeabur.com](https://zeabur.com)
2. **Create Project** → pick the region closest to you
3. **+ Add Service** → **Docker Image** → paste:
   ```
   ghcr.io/mecaca-global-inc/waapi-gateway:latest
   ```
4. **Variables** → paste:
   ```env
   ADMIN_USER=admin
   ADMIN_PASS=ChangeMeToSomethingStrong!23
   CORS_ORIGINS=*
   LOG_FORMAT=json
   DEVICE_NAME=WAAPI Gateway
   ```
5. **Volumes** → Add → ID `storages`, Mount Directory `/app/storages`
6. **Networking** → Generate Domain → port `8080`
7. Open the domain in your browser. Login `admin` / your `ADMIN_PASS`.

That's it. The container ships gateway + dashboard in one binary.

---

## Walkthrough (with screenshots)

### 1 — Sign in

Open [zeabur.com](https://zeabur.com) and sign in with GitHub. You don't need to grant access to any repo — we're pulling the prebuilt Docker image, not source.

### 2 — Create a project

**Create Project** → choose region (Singapore for SEA, Frankfurt for EU, etc.).

### 3 — Add the gateway service

**+ Add Service** → **Docker Image**. Paste:

```
ghcr.io/mecaca-global-inc/waapi-gateway:latest
```

Hit **Deploy**. Zeabur pulls the image and starts the container.

> **Why this image?** It's a multi-arch (amd64 + arm64) build of the gateway + embedded dashboard, published automatically on every release. Source: <https://github.com/mecaca-global-inc/waapi-gateway>.

### 4 — Environment variables

Service → **Variables** tab → paste:

```env
ADMIN_USER=admin
ADMIN_PASS=ChangeMeToSomethingStrong!23
CORS_ORIGINS=*
LOG_LEVEL=info
LOG_FORMAT=json
DEVICE_NAME=WAAPI Gateway
DB_DIALECT=sqlite3
DB_URI=file:/app/storages/gateway.db?_foreign_keys=on
```

**Replace `ADMIN_PASS`** with something only you know. It must not be one of `admin` / `changeme` / `password` / empty — the gateway refuses to boot with weak defaults.

**Do not set `PORT`** — Zeabur injects it, the binary reads it.

### 5 — Persistent volume (critical)

Service → **Volumes** tab → **Add Volume**:

| Field           | Value           |
| --------------- | --------------- |
| Volume ID       | `storages`      |
| Mount Directory | `/app/storages` |

Without this, every redeploy wipes WhatsApp pairings and you'll have to re-scan the QR each time.

### 6 — Public domain

Service → **Networking** → **Generate Domain**:

- Subdomain: anything available, e.g. `waapi`
- Port: **8080** (Zeabur's injected `$PORT`)
- Click **Confirm**

You'll get `https://<your-subdomain>.zeabur.app` with HTTPS.

### 7 — Verify

```bash
curl https://<your-subdomain>.zeabur.app/healthz
# → {"ok":true}
```

Open the URL in a browser. Login screen renders. Use `admin` / your `ADMIN_PASS`.

### 8 — First WhatsApp pairing

1. Dashboard → **Sessions** → enter a name e.g. `main` → **Create**
2. Click the `main` row → **Start**
3. QR appears → scan with phone (WhatsApp → *Linked Devices* → *Link a Device*)
4. Status flips `SCAN_QR` → `WORKING`

### 9 — Stay updated

Zeabur **doesn't auto-pull image updates** by default for Docker Image sources. When a new gateway release ships:

- Service → **Settings** → **Image** → pick a newer tag (e.g. `:1.2.0`) or stay on `:latest`
- Click **Redeploy**

Or pin a specific version (recommended for production):

```
ghcr.io/mecaca-global-inc/waapi-gateway:1.1.0
```

Browse all tags: <https://github.com/mecaca-global-inc/waapi-gateway/pkgs/container/waapi-gateway>

---

## Optional — Custom domain

Service → **Networking** → **Add Domain** → enter `api.yourdomain.com` → Zeabur shows a DNS record → add it at your registrar → HTTPS auto-issued in ~2 min.

After the custom domain is live, tighten CORS:

```env
CORS_ORIGINS=https://api.yourdomain.com,https://yourdomain.com
```

Save → Redeploy.

---

## Plan / cost

| Tier         | Use case                            | Cost                |
| ------------ | ----------------------------------- | ------------------- |
| Hobby/Free   | development, sleeps after inactivity | $0                  |
| Developer    | always-on (required for WhatsApp WS) | ~$5/mo per service  |
| + Volume     | 1 GB SQLite + media                  | included            |

**Always-on plan required for production.** WhatsApp drops the session if the process sleeps.

---

## Troubleshooting

| Symptom                              | Fix                                                                                |
| ------------------------------------ | ---------------------------------------------------------------------------------- |
| 502 from Cloudflare                  | Check **Logs**. Usually weak `ADMIN_PASS` aborting boot.                           |
| QR rescan after redeploy             | Volume not mounted — redo Step 5.                                                  |
| Dashboard 404 on sub-pages           | Old image — pull `:latest` again via Redeploy.                                     |
| `invalid api key` on dashboard       | localStorage stale — click **Logout** then log in again.                           |
| `ADMIN_PASS is weak`                 | Pick a password not on the blocklist, save, redeploy.                              |
| Container restarting in a loop       | **Logs** tab — look at the first error line.                                       |
| Cannot see the image when typing it  | Type the full string. Public package, no auth needed.                              |

---

## For maintainers — alternative: install from Git source

If you want push-to-deploy on your own fork:

1. **+ Add Service** → **GitHub** → select your fork → branch `main`
2. Install the Zeabur GitHub App if prompted
3. Continue from Step 4 above

Every `git push origin main` triggers a rebuild. Slower than the prebuilt image (~2 min per deploy) but lets you customize.
