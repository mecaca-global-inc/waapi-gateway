package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerResources exposes read-only "browseable" resources that an agent
// can subscribe to for fresh context without spending a tool call.
func registerResources(s *server.MCPServer, c *Client) {
	add := func(uri, name, desc, mime string, path string) {
		res := mcp.NewResource(uri, name,
			mcp.WithResourceDescription(desc),
			mcp.WithMIMEType(mime),
		)
		s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			raw, err := c.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      uri,
					MIMEType: mime,
					Text:     string(raw),
				},
			}, nil
		})
	}

	add("waapi://sessions",
		"Sessions",
		"All WhatsApp sessions and their statuses.",
		"application/json",
		"/api/sessions")

	add("waapi://openapi",
		"OpenAPI spec",
		"Full OpenAPI 3.1 specification — agents can self-discover the entire REST surface.",
		"application/yaml",
		"/openapi.yaml")
}
