// Package web serves the static dashboard built by Next.js (`next build` with
// `output: "export"`). The built files are baked into the Go binary at compile
// time via go:embed, so a single binary serves both the REST API and the UI.
//
// During development the embed is empty (the `out/` directory may not exist
// yet); HasAssets reports whether any assets were embedded so the caller can
// skip mounting routes when running outside Docker.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed all:dist
var distFS embed.FS

// fsRoot strips the leading "dist/" prefix so URLs map directly to filenames.
func fsRoot() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return distFS
	}
	return sub
}

// HasAssets reports whether the dashboard was embedded at build time.
func HasAssets() bool {
	entries, err := fs.ReadDir(distFS, "dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// Mount adds dashboard routes to the Fiber app. Static files are served from
// the embedded fs; unknown paths fall back to /index.html so client-side
// routing works after a hard refresh.
func Mount(app *fiber.App) {
	if !HasAssets() {
		return
	}
	root := fsRoot()

	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(root),
		Browse:       false,
		NotFoundFile: "index.html",
		MaxAge:       3600,
		Index:        "index.html",
	}))

	// Hand-rolled fallback for any path not pre-rendered (e.g. nested routes).
	app.Use(func(c *fiber.Ctx) error {
		// API and ws endpoints already matched earlier in the chain; if we
		// reach here it's a dashboard URL that wasn't statically generated.
		p := c.Path()
		if strings.HasPrefix(p, "/api/") || p == "/healthz" || p == "/readyz" ||
			p == "/openapi.yaml" || p == "/openapi.json" || strings.HasPrefix(p, "/ws") {
			return c.Next()
		}
		data, err := fs.ReadFile(root, "index.html")
		if err != nil {
			return c.Next()
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(data)
	})
}
