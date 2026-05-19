package api

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed openapi.yaml
var openAPISpec []byte

// docsRoutes serves only the OpenAPI spec. The interactive UI lives in the
// dashboard at /docs (auth-gated).
func (s *Server) docsRoutes() {
	handler := func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/yaml; charset=utf-8")
		c.Set("Access-Control-Allow-Origin", "*")
		return c.Send(openAPISpec)
	}
	s.app.Get("/openapi.yaml", handler)
	s.app.Get("/openapi.json", handler)
}
