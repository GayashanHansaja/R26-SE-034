package mcp

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/nimendra/ERPBridge/internal/connector"
)

// Resource represents a read-only data source that can be accessed by an AI agent.
type Resource struct {
	// Name is the unique identifier for the resource.
	Name string `json:"name"`
	// Description provides a human-readable explanation of the resource content.
	Description string `json:"description"`
	// URITemplate defines the pattern used to access this resource.
	URITemplate string `json:"uriTemplate"`
	// MimeType specifies the format of the resource content (e.g., "text/markdown").
	MimeType string `json:"mimeType,omitempty"`
	// Endpoint provides the technical details for fetching the resource data.
	Endpoint *Endpoint `json:"endpoint,omitempty"`
}

// Execute fetches the resource content from the underlying ERP system.
func (r *Resource) Execute(ctx context.Context, uri string, conn ERPConnector) (string, error) {
	if r.Endpoint == nil {
		return "", fmt.Errorf("resource %s has no endpoint configuration", r.Name)
	}

	ep := connector.EndpointConfig{
		Method:  "GET",
		Path:    r.Endpoint.Path,
		BaseURL: "",
		Auth: connector.AuthConfig{
			Type:     r.Endpoint.Auth.Type,
			Header:   r.Endpoint.Auth.Header,
			Key:      r.Endpoint.Auth.KeyRef,
			Username: r.Endpoint.Auth.Username,
			Token:    r.Endpoint.Auth.Token,
		},
	}

	// Handle relative paths
	if !strings.HasPrefix(ep.Path, "http") {
		ep.Path = "http://localhost:8081" + ep.Path
	}

	resp, err := conn.Call(ctx, ep, nil, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
