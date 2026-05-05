package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nimendra/ERPBridge/internal/connector"
)

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Module      string      `json:"module,omitempty"`
	InputSchema InputSchema `json:"inputSchema"`
	Endpoint    *Endpoint   `json:"endpoint,omitempty"` // Exported for internal use and saved to JSON
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

type Endpoint struct {
	Method string   `json:"method"`
	Path   string   `json:"path"`
	Auth   AuthInfo `json:"auth"`
}

type AuthInfo struct {
	Type     string `json:"type"`
	Header   string `json:"header"`
	KeyRef   string `json:"keyRef"`   // Reference or actual key
	Username string `json:"username"` // For Basic Auth
	Token    string `json:"token"`    // For Bearer Auth
}

type ToolCallRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolResult struct {
	Result  any  `json:"result"`
	Error   any  `json:"error,omitempty"`
	IsError bool `json:"isError,omitempty"`
}

type ERPConnector interface {
	Call(ctx context.Context, ep connector.EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error)
}

func (t *Tool) Execute(ctx context.Context, args map[string]any, conn ERPConnector) (*ToolResult, error) {
	if t.Endpoint == nil {
		return nil, fmt.Errorf("tool %s has no endpoint configuration", t.Name)
	}

	queryParams := url.Values{}
	var body io.Reader

	if t.Endpoint.Method == "GET" {
		for k, v := range args {
			queryParams.Set(k, fmt.Sprintf("%v", v))
		}
	} else {
		data, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("marshal arguments: %w", err)
		}
		body = strings.NewReader(string(data))
	}

	ep := connector.EndpointConfig{
		Method:  t.Endpoint.Method,
		Path:    t.Endpoint.Path,
		BaseURL: "", // Expected to be full URL or prefixed in connector
		Auth: connector.AuthConfig{
			Type:     t.Endpoint.Auth.Type,
			Header:   t.Endpoint.Auth.Header, // Note: connector uses Header for username in Basic auth
			Key:      t.Endpoint.Auth.KeyRef, // Note: connector uses Key for password in Basic auth
			Username: t.Endpoint.Auth.Username,
			Token:    t.Endpoint.Auth.Token,
		},
	}

	resp, err := conn.Call(ctx, ep, queryParams, body)
	if err != nil {
		return nil, fmt.Errorf("erp call failed: %w", err)
	}
	defer resp.Body.Close()

	var resultData any
	if err := json.NewDecoder(resp.Body).Decode(&resultData); err != nil {
		return nil, fmt.Errorf("decode erp response: %w", err)
	}

	return &ToolResult{
		Result:  resultData,
		IsError: resp.StatusCode >= 400,
	}, nil
}
