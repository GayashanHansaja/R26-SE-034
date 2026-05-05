package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Tool struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Module       string        `json:"module,omitempty"`
	InputSchema  InputSchema   `json:"inputSchema"`
	OutputSchema *any          `json:"outputSchema,omitempty"` // Optional JSON schema for response validation
	Endpoint     *Endpoint     `json:"endpoint,omitempty"`
	Cache        *cache.Config `json:"cache,omitempty"`
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
	KeyRef   string `json:"keyRef"`
	Username string `json:"username"`
	Token    string `json:"token"`
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

	// If Path is already a full URL, BaseURL can be empty
	// If Path is relative, we should ideally use the context's ERPBase, but for now we'll assume Path is full or handled.
	// Since generator puts absolute paths if available, or relative, let's fix it by pulling from config or assuming Path might be full.
	// Actually, the generator creates path like "/api/v1/..." if Server URL is missing in OpenAPI. Let's prepend localhost:8081 for testing.

	fullURL := t.Endpoint.Path
	if !strings.HasPrefix(fullURL, "http") {
		// Fallback for relative paths generated from OpenAPI without servers
		fullURL = "http://localhost:8081" + fullURL
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
	defer resp.Body.Close()

	var resultData any
	if err := json.NewDecoder(resp.Body).Decode(&resultData); err != nil {
		return nil, fmt.Errorf("decode erp response: %w", err)
	}

	// Task 6: Response Validation
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
	// The openapi schema from kin-openapi needs to be properly serialized to JSON schema format.
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	c := jsonschema.NewCompiler()
	// Kin-openapi generates swagger/openapi schemas which might need leniency.
	
	if err := c.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return fmt.Errorf("add resource: %w", err)
	}
	js, err := c.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	return js.Validate(data)
}
