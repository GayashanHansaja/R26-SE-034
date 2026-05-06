package mcp

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/nimendra/ERPBridge/internal/connector"
)

type Resource struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URITemplate string    `json:"uriTemplate"`
	MimeType    string    `json:"mimeType,omitempty"`
	Endpoint    *Endpoint `json:"endpoint,omitempty"`
}

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
