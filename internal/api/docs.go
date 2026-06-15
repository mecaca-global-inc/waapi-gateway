package api

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

//go:embed openapi.yaml
var openAPISpec []byte

// serversMarker is the literal block in openapi.yaml that we replace at
// serve-time with a servers list derived from the incoming request. Keep this
// in sync with the top of openapi.yaml.
var serversMarker = []byte("servers:\n  - url: http://localhost:3000\n    description: Local")

// docsRoutes serves the OpenAPI spec. The interactive UI lives in the
// dashboard at /docs (auth-gated). The `servers:` block is rewritten on each
// request so Swagger UI's "Try it out" targets whatever host actually served
// the page (production domain, staging, localhost) — no rebuild needed.
func (s *Server) docsRoutes() {
	handler := func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml; charset=utf-8")
		c.Set("Access-Control-Allow-Origin", "*")
		return c.Send(specForRequest(c))
	}
	s.app.Get("/openapi.yaml", handler)
	s.app.Get("/openapi.json", handler)
}

// specForRequest returns the embedded spec with its `servers:` block rewritten
// to put the request's own origin first, falling back to localhost for local
// dev convenience.
func specForRequest(c *fiber.Ctx) []byte {
	scheme := c.Protocol() // honours X-Forwarded-Proto behind proxies
	host := c.Hostname()   // honours X-Forwarded-Host
	if host == "" {
		return openAPISpec
	}
	origin := fmt.Sprintf("%s://%s", scheme, host)

	var servers string
	if host == "localhost" || host == "127.0.0.1" {
		servers = fmt.Sprintf("servers:\n  - url: %s\n    description: This server", origin)
	} else {
		servers = fmt.Sprintf(
			"servers:\n  - url: %s\n    description: This server\n  - url: http://localhost:3000\n    description: Local dev",
			origin,
		)
	}
	return bytes.Replace(openAPISpec, serversMarker, []byte(servers), 1)
}
