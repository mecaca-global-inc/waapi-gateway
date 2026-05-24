// Package mcp exposes the WAAPI Gateway over the Model Context Protocol.
//
// The MCP server is a thin adapter: it forwards every tool invocation to the
// gateway's existing REST API. This avoids duplicating business logic between
// HTTP handlers and MCP tools. Configure the target gateway URL and API key
// via env:
//
//	WAAPI_GATEWAY_URL   default http://localhost:3000
//	WAAPI_API_KEY       required
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client wraps the gateway REST API for MCP tools to call.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient() (*Client, error) {
	base := strings.TrimRight(os.Getenv("WAAPI_GATEWAY_URL"), "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	if _, err := url.Parse(base); err != nil {
		return nil, fmt.Errorf("invalid WAAPI_GATEWAY_URL: %w", err)
	}
	key := os.Getenv("WAAPI_API_KEY")
	if key == "" {
		return nil, errors.New("WAAPI_API_KEY env var is required (POST /api/login to obtain one)")
	}
	return &Client{
		baseURL: base,
		apiKey:  key,
		http:    &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// do executes an authenticated request and decodes the response into out (or
// returns the raw body bytes if out is nil).
func (c *Client) do(ctx context.Context, method, path string, body any, out any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("%s %s: %s — %s", method, path, resp.Status, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return raw, fmt.Errorf("decode %s %s: %w", method, path, err)
		}
	}
	return raw, nil
}

func (c *Client) Get(ctx context.Context, path string, out any) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil, out)
}
func (c *Client) Post(ctx context.Context, path string, body any, out any) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, body, out)
}
func (c *Client) Put(ctx context.Context, path string, body any, out any) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, body, out)
}
func (c *Client) Delete(ctx context.Context, path string, out any) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, path, nil, out)
}
