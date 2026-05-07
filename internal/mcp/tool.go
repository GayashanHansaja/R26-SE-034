package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Tool represents a protocol-compliant MCP tool that an AI agent can invoke.
type Tool struct {
	// Name is the unique identifier for the tool.
	Name string `json:"name"`
	// Description provides a human-readable explanation of what the tool does.
	Description string `json:"description"`
	// Module is the ERP module this tool belongs to (e.g., "finance").
	Module string `json:"module,omitempty"`
	// InputSchema defines the expected arguments for the tool call.
	InputSchema InputSchema `json:"inputSchema"`
	// OutputSchema is an optional JSON schema used for validating the ERP response.
	OutputSchema *any `json:"outputSchema,omitempty"`
	// Endpoint contains the connection details for the underlying ERP API.
	Endpoint *Endpoint `json:"endpoint,omitempty"`
	// Cache defines the caching strategy for this tool.
	Cache *cache.Config `json:"cache,omitempty"`
}

// InputSchema defines the structure of arguments required by a tool.
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property describes a single field in a tool's input schema.
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// Endpoint provides the technical configuration for an ERP API call.
type Endpoint struct {
	Method string   `json:"method"`
	Path   string   `json:"path"`
	Auth   AuthInfo `json:"auth"`
}

// AuthInfo contains the credentials and authentication method for an endpoint.
type AuthInfo struct {
	Type     string `json:"type"`
	Header   string `json:"header"`
	KeyRef   string `json:"keyRef"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// ToolCallRequest represents an incoming request from an MCP client to invoke a tool.
type ToolCallRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolResult encapsulates the outcome of a tool invocation.
type ToolResult struct {
	// Result contains the successful response data from the ERP.
	Result any `json:"result"`
	// Error contains details about a failed invocation.
	Error any `json:"error,omitempty"`
	// IsError indicates if the invocation failed.
	IsError bool `json:"isError,omitempty"`
}

// ERPConnector defines the interface for executing calls to external ERP systems.
type ERPConnector interface {
	Call(ctx context.Context, ep connector.EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error)
}

// Execute performs the actual tool invocation by calling the underlying ERP API.
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

	fullURL := t.Endpoint.Path
	if !strings.HasPrefix(fullURL, "http") {
		// Fallback for relative paths
		baseURL := os.Getenv("ERP_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8081"
		}
		fullURL = baseURL + fullURL
	}

	ep := connector.EndpointConfig{
		Method:  t.Endpoint.Method,
		Path:    fullURL,
		BaseURL: "",
		Auth: connector.AuthConfig{
			Type:     t.Endpoint.Auth.Type,
			Header:   t.Endpoint.Auth.Header,
			Key:      t.Endpoint.Auth.KeyRef,
			Username: t.Endpoint.Auth.Username,
			Token:    t.Endpoint.Auth.Token,
		},
	}

	resp, err := conn.Call(ctx, ep, queryParams, body)
	if err != nil {
		return nil, fmt.Errorf("erp call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var resultData any
	if err := json.NewDecoder(resp.Body).Decode(&resultData); err != nil {
		return nil, fmt.Errorf("decode erp response: %w", err)
	}

	if t.OutputSchema != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := validateResponse(resultData, t.OutputSchema); err != nil {
			return &ToolResult{
				Result:  resultData,
				Error:   fmt.Sprintf("response validation failed: %v", err),
				IsError: true,
			}, nil
		}
	}

	return &ToolResult{
		Result:  resultData,
		IsError: resp.StatusCode >= 400,
	}, nil
}

func validateResponse(data any, schema any) error {
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("add resource: %w", err)
	}
	js, err := c.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	return js.Validate(data)
}
