# MCP Server

WAAPI Gateway ships a **Model Context Protocol** server in the same binary. Any MCP-compatible agent — Claude Desktop, Cursor, OpenCode, Codex, n8n with MCP nodes, custom agent runtimes — can drive WhatsApp via 45+ typed tools and a few read-only resources.

The MCP server is an adapter: it forwards every tool invocation to the gateway's existing REST API. It does **not** open WhatsApp sessions on its own — you still run the gateway (locally or on Zeabur) and point the MCP server at it.

---

## 1. Get an API key

```bash
# replace with your gateway URL + admin password
curl -X POST https://waapi.zeabur.app/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<your password>"}'
# → {"api_key":"8604c43f-d6b3-4305-b77f-..."}
```

Save the `api_key`. You'll set it as `WAAPI_API_KEY`.

---

## 2. Run modes

### stdio (local agents — Claude Desktop, Cursor, OpenCode, Codex)

The agent spawns the MCP server as a subprocess and talks to it over stdin/stdout.

Two ways to launch:

#### A) Direct binary (you've downloaded the gateway)
```bash
WAAPI_GATEWAY_URL=https://waapi.zeabur.app \
WAAPI_API_KEY=... \
waapi-gateway mcp
```

#### B) Docker (one-shot subprocess — no local install)
```bash
docker run -i --rm \
  -e WAAPI_GATEWAY_URL=https://waapi.zeabur.app \
  -e WAAPI_API_KEY=... \
  ghcr.io/mecaca-global-inc/waapi-gateway:latest mcp
```

### HTTP + SSE (remote agents, hosted gateways)

Expose the MCP server on a port so cloud agents can connect over HTTPS:

```bash
WAAPI_GATEWAY_URL=https://waapi.zeabur.app \
WAAPI_API_KEY=... \
MCP_BEARER_TOKEN=... \
waapi-gateway mcp --http :3001
```

Endpoints:
- `GET  /sse`     — SSE event stream
- `POST /message` — client→server messages

Both require `Authorization: Bearer <MCP_BEARER_TOKEN>`. If `MCP_BEARER_TOKEN` is unset, the gateway falls back to `WAAPI_API_KEY` so one secret covers both directions.

---

## 3. Client configuration snippets

### Claude Desktop
File: `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows).

```json
{
  "mcpServers": {
    "waapi": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "WAAPI_GATEWAY_URL=https://waapi.zeabur.app",
        "-e", "WAAPI_API_KEY=YOUR_API_KEY",
        "ghcr.io/mecaca-global-inc/waapi-gateway:latest", "mcp"
      ]
    }
  }
}
```

Restart Claude Desktop. New chats expose tools prefixed `waapi_…`.

### Cursor
File: `~/.cursor/mcp.json` — same shape as Claude Desktop.

### OpenCode / Codex
Same `mcpServers` block, dropped into the project's config (see your tool's docs for the exact filename).

### Remote (any MCP client that supports SSE)

```json
{
  "mcpServers": {
    "waapi": {
      "url": "https://mcp.waapi.link/sse",
      "headers": { "Authorization": "Bearer YOUR_MCP_BEARER_TOKEN" }
    }
  }
}
```

### n8n with the MCP Client node

In n8n's HTTP / community MCP node, set:
- URL: `https://mcp.waapi.link/sse`
- Auth: bearer `YOUR_MCP_BEARER_TOKEN`

---

## 4. Available tools (45)

| Group | Tools |
|---|---|
| Sessions | `waapi_sessions_list`, `_get`, `_create`, `_start`, `_stop`, `_logout`, `_delete` |
| Auth | `waapi_auth_qr`, `waapi_auth_request_code` |
| Send | `waapi_send_text`, `_image`, `_video`, `_voice`, `_file`, `_location`, `_contact`, `_seen`, `waapi_start_typing`, `waapi_stop_typing` |
| Read | `waapi_me`, `waapi_contacts_list`, `waapi_chats_list` |
| Groups | `waapi_groups_list`, `_create`, `_join`, `_get_info`, `_leave`, `_add_participants`, `_remove_participants`, `_promote_participants`, `_demote_participants`, `_set_name`, `_set_topic`, `_set_locked`, `_set_announce`, `_set_photo`, `_set_disappearing`, `_get_invite_link`, `_revoke_invite_link` |
| Webhooks | `waapi_webhooks_list`, `_add`, `_delete` |
| API keys | `waapi_keys_list`, `_create`, `_delete` |

Restrict the exposed surface with `--tools=`:
```bash
waapi-gateway mcp --tools=waapi_send_text,waapi_send_image,waapi_groups_list
```

## 5. Resources

| URI | What |
|---|---|
| `waapi://sessions` | Live JSON list of sessions + status |
| `waapi://openapi`  | Full OpenAPI 3.1 spec — let the agent self-discover the REST surface |

Agents can subscribe to a resource and re-read it without spending tool calls.

---

## 6. Verify it works

Once your client is configured, ask the agent:
> List my WhatsApp sessions.

It should call `waapi_sessions_list` and return your sessions. Then:
> Send "hello from the agent" to 6281234567890 on session "main".

Calls `waapi_send_text` → message arrives on the phone.

## 7. Security notes

- The MCP server has the **full power of an admin API key**. Treat the key like a password.
- For HTTP+SSE mode, **always** use HTTPS in front (Zeabur/Cloudflare/Caddy/nginx). Bearer tokens over plaintext are exposed to anyone on the path.
- In production, issue a dedicated API key for MCP via `Settings → API keys` on the dashboard — rotate it independently from your other integrations.
- Consider running MCP behind a private network or VPN if you don't need public agent access.

---

## 8. Troubleshooting

| Symptom | Fix |
|---|---|
| `WAAPI_API_KEY env var is required` | Export the env or set it in the client config. |
| Tools list is empty in client | Restart the agent app; some clients cache the tool list at startup. |
| `Bearer` rejected on HTTP mode | Check `MCP_BEARER_TOKEN` matches the header value the client sends. |
| Tool call returns `401 Unauthorized` | API key is wrong / revoked. Issue a new one in the dashboard. |
| Tool call returns `503 session not connected` | Start the session: `waapi_sessions_start` then `waapi_auth_qr`. |
| Image / file send fails | The gateway needs a **publicly reachable** URL; localhost won't work. |

---

Source: <https://github.com/mecaca-global-inc/waapi-gateway/blob/main/internal/mcp>
