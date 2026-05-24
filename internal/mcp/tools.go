package mcp

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTools wires every WAAPI REST endpoint as an MCP tool. Tools forward
// to the gateway HTTP API using the shared Client. The allow list (--tools)
// is honoured here — empty means all.
func registerTools(s *server.MCPServer, c *Client, allow []string) {
	allowed := func(name string) bool {
		if len(allow) == 0 {
			return true
		}
		for _, a := range allow {
			if a == name {
				return true
			}
		}
		return false
	}

	for _, t := range allTools(c) {
		if !allowed(t.tool.Name) {
			continue
		}
		s.AddTool(t.tool, t.handler)
	}
}

type toolDef struct {
	tool    mcp.Tool
	handler server.ToolHandlerFunc
}

// allTools returns every tool definition. Grouped by REST category for
// readability; tool names follow waapi_<category>_<action>.
func allTools(c *Client) []toolDef {
	out := []toolDef{}
	out = append(out, sessionsTools(c)...)
	out = append(out, authTools(c)...)
	out = append(out, messageTools(c)...)
	out = append(out, readTools(c)...)
	out = append(out, groupTools(c)...)
	out = append(out, webhookTools(c)...)
	out = append(out, keyTools(c)...)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

// rawTool builds a tool whose handler proxies an HTTP call and returns the
// raw JSON body (or an error). Use for simple passthroughs.
func rawTool(name, desc, method, pathTpl string, c *Client, opts ...mcp.ToolOption) toolDef {
	opts = append([]mcp.ToolOption{mcp.WithDescription(desc)}, opts...)
	return toolDef{
		tool: mcp.NewTool(name, opts...),
		handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			path, err := renderPath(pathTpl, req)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			body := req.GetArguments() // may include extra fields
			body = stripPathParams(body, pathTpl)
			raw, err := c.do(ctx, method, path, bodyOrNil(method, body), nil)
			if err != nil {
				if len(raw) > 0 {
					return mcp.NewToolResultError(string(raw)), nil
				}
				return mcp.NewToolResultErrorFromErr("call failed", err), nil
			}
			if len(raw) == 0 {
				return mcp.NewToolResultText("{}"), nil
			}
			return mcp.NewToolResultText(string(raw)), nil
		},
	}
}

// renderPath substitutes {param} placeholders in pathTpl with URL-escaped
// values from the request arguments.
func renderPath(pathTpl string, req mcp.CallToolRequest) (string, error) {
	args := req.GetArguments()
	out := pathTpl
	for k, v := range args {
		placeholder := "{" + k + "}"
		if !contains(out, placeholder) {
			continue
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("path param %q must be a string", k)
		}
		out = replaceAll(out, placeholder, url.PathEscape(s))
	}
	if firstBrace(out) >= 0 {
		return "", fmt.Errorf("missing path parameter in %s", pathTpl)
	}
	return out, nil
}

// stripPathParams removes keys that were consumed by the URL template so they
// don't get re-sent inside the JSON body.
func stripPathParams(args map[string]any, pathTpl string) map[string]any {
	if len(args) == 0 {
		return args
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		if contains(pathTpl, "{"+k+"}") {
			continue
		}
		out[k] = v
	}
	return out
}

func bodyOrNil(method string, body map[string]any) any {
	if method == "GET" || method == "DELETE" || len(body) == 0 {
		return nil
	}
	return body
}

// tiny string helpers (avoid an extra import)
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			return out + s
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
func firstBrace(s string) int {
	for i, r := range s {
		if r == '{' {
			return i
		}
	}
	return -1
}

// ─────────────────────────────────────────────────────────────────────────────
// Sessions
// ─────────────────────────────────────────────────────────────────────────────

func sessionsTools(c *Client) []toolDef {
	sessionName := mcp.WithString("session",
		mcp.Required(),
		mcp.Description("Session name (e.g. \"default\")."),
	)
	return []toolDef{
		rawTool("waapi_sessions_list",
			"List every WhatsApp session and its current status.",
			"GET", "/api/sessions", c),

		rawTool("waapi_sessions_get",
			"Get a single session's status and JID.",
			"GET", "/api/sessions/{session}", c, sessionName),

		rawTool("waapi_sessions_create",
			"Create a new session. After creation, call waapi_sessions_start then waapi_auth_qr.",
			"POST", "/api/sessions", c,
			mcp.WithString("name", mcp.Required(), mcp.Description("Session name (unique, lowercase recommended).")),
		),

		rawTool("waapi_sessions_start",
			"Connect the session to WhatsApp servers. Triggers QR or auto-resume.",
			"POST", "/api/sessions/{session}/start", c, sessionName),

		rawTool("waapi_sessions_stop",
			"Disconnect the session but keep credentials. Use waapi_sessions_start to resume.",
			"POST", "/api/sessions/{session}/stop", c, sessionName),

		rawTool("waapi_sessions_logout",
			"Log the session out of WhatsApp. Requires re-pairing.",
			"POST", "/api/sessions/{session}/logout", c, sessionName),

		rawTool("waapi_sessions_delete",
			"Delete the session (logout + remove from DB).",
			"DELETE", "/api/sessions/{session}", c, sessionName),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth (QR + pairing code)
// ─────────────────────────────────────────────────────────────────────────────

func authTools(c *Client) []toolDef {
	sessionPath := mcp.WithString("session", mcp.Required(),
		mcp.Description("Session name."))
	return []toolDef{
		rawTool("waapi_auth_qr",
			"Get the current QR code string for a session (raw text — render it as a QR for the user to scan).",
			"GET", "/api/{session}/auth/qr", c, sessionPath),

		rawTool("waapi_auth_request_code",
			"Request an 8-digit pairing code for a phone number (alternative to QR).",
			"POST", "/api/{session}/auth/request-code", c, sessionPath,
			mcp.WithString("phone", mcp.Required(),
				mcp.Description("Phone in E.164 form WITHOUT '+' (e.g. \"6281234567890\").")),
		),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Messaging
// ─────────────────────────────────────────────────────────────────────────────

func messageTools(c *Client) []toolDef {
	session := mcp.WithString("session", mcp.Required(),
		mcp.Description("Session name."))
	chatID := mcp.WithString("chat_id", mcp.Required(),
		mcp.Description("Destination JID or bare phone. Examples: \"6281234567890\", \"628...@s.whatsapp.net\", \"...@lid\", \"1234567890-1700000000@g.us\"."))
	expiration := mcp.WithNumber("expiration",
		mcp.Description("Disappearing-messages timer in seconds. Omit to auto-inherit chat's timer. Pass -1 to force non-ephemeral."))
	caption := mcp.WithString("caption", mcp.Description("Optional caption."))
	url := mcp.WithString("url", mcp.Required(), mcp.Description("Publicly reachable URL for the media to send."))
	mimetype := mcp.WithString("mimetype", mcp.Description("Override mime type."))

	return []toolDef{
		rawTool("waapi_send_text",
			"Send a text message.",
			"POST", "/api/sendText", c,
			session, chatID,
			mcp.WithString("text", mcp.Required(), mcp.Description("Message body.")),
			expiration,
		),

		rawTool("waapi_send_image",
			"Send an image by URL.",
			"POST", "/api/sendImage", c,
			session, chatID, url, caption, mimetype, expiration,
		),

		rawTool("waapi_send_video",
			"Send a video by URL.",
			"POST", "/api/sendVideo", c,
			session, chatID, url, caption, mimetype, expiration,
		),

		rawTool("waapi_send_voice",
			"Send a voice note (PTT) by URL. Prefer audio/ogg; codecs=opus.",
			"POST", "/api/sendVoice", c,
			session, chatID, url, mimetype, expiration,
		),

		rawTool("waapi_send_file",
			"Send a document/file by URL.",
			"POST", "/api/sendFile", c,
			session, chatID, url, caption, mimetype, expiration,
			mcp.WithString("filename", mcp.Description("Filename shown to the recipient.")),
		),

		rawTool("waapi_send_location",
			"Send a geographic pin.",
			"POST", "/api/sendLocation", c,
			session, chatID,
			mcp.WithNumber("latitude", mcp.Required()),
			mcp.WithNumber("longitude", mcp.Required()),
			mcp.WithString("name", mcp.Description("Place name.")),
			mcp.WithString("address", mcp.Description("Place address.")),
			expiration,
		),

		rawTool("waapi_send_contact",
			"Send a vCard contact.",
			"POST", "/api/sendContact", c,
			session, chatID,
			mcp.WithString("display_name", mcp.Description("Contact display name.")),
			mcp.WithString("vcard", mcp.Required(),
				mcp.Description("Full vCard 3.0 text. Example: \"BEGIN:VCARD\\nVERSION:3.0\\nFN:John\\nTEL;type=CELL:+6281234567890\\nEND:VCARD\".")),
			expiration,
		),

		rawTool("waapi_send_seen",
			"Mark messages as read (blue ticks).",
			"POST", "/api/sendSeen", c,
			session, chatID,
			mcp.WithArray("message_ids", mcp.Required(),
				mcp.Items(map[string]any{"type": "string"}),
				mcp.Description("Message IDs to mark read.")),
			mcp.WithString("sender", mcp.Description("Required only for group reads.")),
		),

		rawTool("waapi_start_typing",
			"Show 'typing…' indicator in the chat.",
			"POST", "/api/startTyping", c, session, chatID),

		rawTool("waapi_stop_typing",
			"Clear the typing indicator.",
			"POST", "/api/stopTyping", c, session, chatID),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Read (me / contacts / chats)
// ─────────────────────────────────────────────────────────────────────────────

func readTools(c *Client) []toolDef {
	session := mcp.WithString("session", mcp.Required(),
		mcp.Description("Session name."))
	return []toolDef{
		rawTool("waapi_me", "Get the logged-in account's profile (JID, push name, platform).",
			"GET", "/api/{session}/me", c, session),
		rawTool("waapi_contacts_list", "List known contacts for the session.",
			"GET", "/api/{session}/contacts", c, session),
		rawTool("waapi_chats_list", "List recent chats (aliases contacts in v1).",
			"GET", "/api/{session}/chats", c, session),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Groups
// ─────────────────────────────────────────────────────────────────────────────

func groupTools(c *Client) []toolDef {
	session := mcp.WithString("session", mcp.Required(),
		mcp.Description("Session name."))
	gid := mcp.WithString("gid", mcp.Required(),
		mcp.Description("Group JID (e.g. \"1234567890-1700000000@g.us\") or bare group ID."))
	participants := mcp.WithArray("participants", mcp.Required(),
		mcp.Items(map[string]any{"type": "string"}),
		mcp.Description("Phone numbers or full JIDs."))

	return []toolDef{
		rawTool("waapi_groups_list",
			"List all groups the session is a member of.",
			"GET", "/api/{session}/groups", c, session),

		rawTool("waapi_groups_create",
			"Create a new group.",
			"POST", "/api/{session}/groups", c, session,
			mcp.WithString("name", mcp.Required(), mcp.Description("Group name (≤25 chars).")),
			participants,
		),

		rawTool("waapi_groups_join",
			"Join a group via invite link or invite code.",
			"POST", "/api/{session}/groups/join", c, session,
			mcp.WithString("link", mcp.Required(),
				mcp.Description("Full invite link or just the code.")),
		),

		rawTool("waapi_groups_get_info",
			"Get full group info (members, settings, owner).",
			"GET", "/api/{session}/groups/{gid}", c, session, gid),

		rawTool("waapi_groups_leave",
			"Leave a group.",
			"POST", "/api/{session}/groups/{gid}/leave", c, session, gid),

		rawTool("waapi_groups_add_participants",
			"Add participants.",
			"POST", "/api/{session}/groups/{gid}/participants/add", c, session, gid, participants),

		rawTool("waapi_groups_remove_participants",
			"Remove participants.",
			"POST", "/api/{session}/groups/{gid}/participants/remove", c, session, gid, participants),

		rawTool("waapi_groups_promote_participants",
			"Promote participants to admin.",
			"POST", "/api/{session}/groups/{gid}/participants/promote", c, session, gid, participants),

		rawTool("waapi_groups_demote_participants",
			"Demote admins back to participants.",
			"POST", "/api/{session}/groups/{gid}/participants/demote", c, session, gid, participants),

		rawTool("waapi_groups_set_name",
			"Rename a group (≤25 chars).",
			"PUT", "/api/{session}/groups/{gid}/name", c, session, gid,
			mcp.WithString("name", mcp.Required())),

		rawTool("waapi_groups_set_topic",
			"Set the group topic / description.",
			"PUT", "/api/{session}/groups/{gid}/topic", c, session, gid,
			mcp.WithString("topic", mcp.Required())),

		rawTool("waapi_groups_set_locked",
			"Restrict group info editing to admins only.",
			"PUT", "/api/{session}/groups/{gid}/locked", c, session, gid,
			mcp.WithBoolean("value", mcp.Required())),

		rawTool("waapi_groups_set_announce",
			"Announce-only mode (only admins can send messages).",
			"PUT", "/api/{session}/groups/{gid}/announce", c, session, gid,
			mcp.WithBoolean("value", mcp.Required())),

		rawTool("waapi_groups_set_photo",
			"Set the group photo from a URL.",
			"PUT", "/api/{session}/groups/{gid}/photo", c, session, gid,
			mcp.WithString("url", mcp.Required(),
				mcp.Description("Publicly reachable image URL (JPEG, ~640px square works best)."))),

		rawTool("waapi_groups_set_disappearing",
			"Set the disappearing-messages timer for a group.",
			"PUT", "/api/{session}/groups/{gid}/disappearing", c, session, gid,
			mcp.WithNumber("seconds", mcp.Required(),
				mcp.Description("Allowed: 0 (off), 86400 (24h), 604800 (7d), 7776000 (90d)."))),

		rawTool("waapi_groups_get_invite_link",
			"Get the current invite link for a group.",
			"GET", "/api/{session}/groups/{gid}/invite-link", c, session, gid),

		rawTool("waapi_groups_revoke_invite_link",
			"Revoke and regenerate the invite link.",
			"POST", "/api/{session}/groups/{gid}/invite-link/revoke", c, session, gid),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Webhooks
// ─────────────────────────────────────────────────────────────────────────────

func webhookTools(c *Client) []toolDef {
	session := mcp.WithString("session", mcp.Required(),
		mcp.Description("Session name."))
	return []toolDef{
		rawTool("waapi_webhooks_list",
			"List webhooks registered for a session.",
			"GET", "/api/{session}/webhooks", c, session),

		rawTool("waapi_webhooks_add",
			"Register a new webhook URL for a session.",
			"POST", "/api/{session}/webhooks", c, session,
			mcp.WithString("url", mcp.Required(),
				mcp.Description("Endpoint that will receive POST events.")),
			mcp.WithString("secret", mcp.Description("HMAC-SHA256 secret (optional).")),
			mcp.WithArray("events",
				mcp.Items(map[string]any{"type": "string"}),
				mcp.Description("Whitelist of events. Empty = all.")),
			mcp.WithBoolean("enabled",
				mcp.Description("Default true.")),
		),

		rawTool("waapi_webhooks_delete",
			"Delete a webhook by ID.",
			"DELETE", "/api/webhooks/{id}", c,
			mcp.WithString("id", mcp.Required(), mcp.Description("Webhook ID returned by waapi_webhooks_list."))),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// API keys
// ─────────────────────────────────────────────────────────────────────────────

func keyTools(c *Client) []toolDef {
	return []toolDef{
		rawTool("waapi_keys_list",
			"List API keys (hashes only — never the plaintext).",
			"GET", "/api/keys", c),

		rawTool("waapi_keys_create",
			"Create a new API key. Plaintext is returned ONCE.",
			"POST", "/api/keys", c,
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Human-readable label, e.g. \"n8n\" or \"laptop\"."))),

		rawTool("waapi_keys_delete",
			"Revoke an API key by ID.",
			"DELETE", "/api/keys/{id}", c,
			mcp.WithString("id", mcp.Required())),
	}
}
