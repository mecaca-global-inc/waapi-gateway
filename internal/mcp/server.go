package mcp

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

// Run boots the MCP server. Called from cmd/server/main.go when the user
// invokes `waapi-gateway mcp [flags]`. The first non-flag argument is
// reserved for future verbs; today flags carry all the config.
func Run(args []string) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	httpAddr := fs.String("http", "", "Serve MCP over HTTP+SSE on this addr (e.g. :3001). Empty = stdio mode.")
	allowedTools := fs.String("tools", "", "Comma-separated list of tool names to expose (empty = all).")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, err := NewClient()
	if err != nil {
		return err
	}

	srv := server.NewMCPServer(
		"waapi-gateway",
		version(),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	registerTools(srv, client, splitCSV(*allowedTools))
	registerResources(srv, client)

	if *httpAddr == "" {
		fmt.Fprintln(os.Stderr, "waapi-gateway MCP server listening on stdio")
		return server.ServeStdio(srv)
	}

	fmt.Fprintf(os.Stderr, "waapi-gateway MCP server listening on http://%s (SSE)\n", *httpAddr)
	sse := server.NewSSEServer(srv,
		server.WithBaseURL(envOr("MCP_PUBLIC_URL", "")),
	)
	mux := http.NewServeMux()
	mux.Handle("/sse", bearerAuth(sse.SSEHandler()))
	mux.Handle("/message", bearerAuth(sse.MessageHandler()))
	return http.ListenAndServe(*httpAddr, mux)
}

func version() string {
	if v := os.Getenv("WAAPI_VERSION"); v != "" {
		return v
	}
	return "dev"
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// bearerAuth gates the HTTP+SSE transport with a per-request bearer token.
// The token must match MCP_BEARER_TOKEN (set this to the gateway API key, or
// a separate key dedicated to MCP traffic).
func bearerAuth(next http.Handler) http.Handler {
	expected := os.Getenv("MCP_BEARER_TOKEN")
	if expected == "" {
		// Fall back to WAAPI_API_KEY so a single env covers both directions.
		expected = os.Getenv("WAAPI_API_KEY")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expected == "" {
			http.Error(w, "MCP_BEARER_TOKEN not configured", http.StatusInternalServerError)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != expected {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

