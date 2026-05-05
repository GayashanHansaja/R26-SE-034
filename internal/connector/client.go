package connector

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type AuthConfig struct {
	Type     string // "api-key" | "basic" | "bearer"
	Header   string // e.g. "X-API-Key"
	Key      string // value or resolved secret
	Username string
	Token    string
}

type EndpointConfig struct {
	Method  string
	Path    string
	BaseURL string
	Auth    AuthConfig
}

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// Call executes an outbound request to the ERP endpoint.
// queryParams are appended to the URL; body is passed as-is for POST/PATCH.
func (c *Client) Call(ctx context.Context, ep EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error) {
	target := ep.BaseURL + ep.Path
	if len(queryParams) > 0 {
		target += "?" + queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, target, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	// Apply auth
	switch ep.Auth.Type {
	case "api-key":
		req.Header.Set(ep.Auth.Header, ep.Auth.Key)
	case "basic":
		user := ep.Auth.Username
		if user == "" {
			user = ep.Auth.Header // Fallback to Header field for username if Username is empty
		}
		req.SetBasicAuth(user, ep.Auth.Key)
	case "bearer":
		token := ep.Auth.Token
		if token == "" {
			token = ep.Auth.Key // Fallback to Key field if Token is empty
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}
