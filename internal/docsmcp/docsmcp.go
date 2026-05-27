// Package docsmcp is a minimal Streamable-HTTP JSON-RPC client for the public
// Vobiz docs MCP server at https://docs.vobiz.ai/mcp. It supports only the
// `search` and `fetch` tools, which is all the CLI uses.
package docsmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

type Client struct {
	endpoint string
	http     *http.Client
	id       atomic.Int64
}

const DefaultEndpoint = "https://docs.vobiz.ai/mcp"

func New(endpoint string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

type Result struct {
	Title   string `json:"title"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
}

func (c *Client) Search(ctx context.Context, query string) ([]Result, error) {
	raw, err := c.callTextTool(ctx, "search", map[string]any{"query": query})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Results []Result `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("docsmcp: decode search payload: %w", err)
	}
	return payload.Results, nil
}

func (c *Client) Fetch(ctx context.Context, path string) (string, error) {
	return c.callTextTool(ctx, "fetch", map[string]any{"path": path})
}

// ---- JSON-RPC plumbing ----

type mcpRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Method  string        `json:"method"`
	Params  mcpToolParams `json:"params"`
}

type mcpToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Result  struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) callTextTool(ctx context.Context, name string, args map[string]any) (string, error) {
	body := mcpRequest{
		JSONRPC: "2.0",
		ID:      c.id.Add(1),
		Method:  "tools/call",
		Params:  mcpToolParams{Name: name, Arguments: args},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docsmcp: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out mcpResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("docsmcp: decode response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("docsmcp: %s (code %d)", out.Error.Message, out.Error.Code)
	}
	if len(out.Result.Content) == 0 {
		return "", fmt.Errorf("docsmcp: empty content from tool %q", name)
	}
	return out.Result.Content[0].Text, nil
}
