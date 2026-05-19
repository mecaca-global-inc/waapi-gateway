# Security

## Reporting a vulnerability

Email **security@waapi.link** with:

- A description of the issue
- Steps to reproduce
- Expected vs. actual behaviour
- Your name / handle for credit (optional)

You'll receive an acknowledgement within 72 hours.

## Hardening checklist for self-hosters

1. **Never use the default admin credentials.** The gateway refuses to boot if `ADMIN_PASS` is one of `admin`, `changeme`, `password`, or empty. Use `ALLOW_WEAK_AUTH=1` only for local development.
2. **Set explicit `CORS_ORIGINS`** in production — not `*`.
3. **Terminate TLS** at a reverse proxy (Caddy, nginx, Traefik). The gateway itself ships plain HTTP.
4. **Restrict the network**: bind the gateway to a private interface or use a firewall — do not expose `:3000` directly to the public internet unless behind WAF/rate-limit.
5. **Rotate API keys** via the dashboard after issuing them; revoke unused ones.
6. **Use a webhook `secret`** for HMAC signing and verify `X-Webhook-Signature` in your receiver before trusting payloads.
7. **Back up `storages/`** — both whatsmeow's device state and the app DB live there.

## Threat model (out of scope)

- WhatsApp itself banning a paired number — that's a WhatsApp ToS decision, not a gateway bug.
- Compromise of the host machine — anyone with root on the server has the SQLite file.
