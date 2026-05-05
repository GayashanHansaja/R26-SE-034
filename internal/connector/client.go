package connector

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/nimendra/ERPBridge/internal/logger"
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
	log  *slog.Logger
}

func NewClient(rootLog *slog.Logger) *Client {
	return &Client{
		http: &http.Client{Timeout: 10 * time.Second},
		log:  logger.Component(rootLog, "connector"),
	}
}

// Call executes an outbound request to the ERP endpoint.
// queryParams are appended to the URL; body is passed as-is for POST/PATCH.
func (c *Client) Call(ctx context.Context, ep EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error) {
	start := time.Now()
	log := logger.FromContext(ctx)

	target := ep.BaseURL + ep.Path
	if len(queryParams) > 0 {
		target += "?" + queryParams.Encode()
	}

	log.Info("erp request",
		slog.String("method", ep.Method),
		slog.String("path", ep.Path),
		slog.String("auth_type", ep.Auth.Type),
	)

	// DEBUG: Log request body
	if body != nil {
		bodyBytes, _ := io.ReadAll(body)
		body = bytes.NewReader(bodyBytes)
		log.Debug("erp request body", slog.String("body", logger.Body(string(bodyBytes))))
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, target, body)
	if err != nil {
		log.Error("erp request failed", slog.String("path", ep.Path), slog.String("error", err.Error()))
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
	resp, err := c.http.Do(req)
	if err != nil {
		log.Error("erp request failed", slog.String("path", ep.Path), slog.String("error", err.Error()))
		return nil, err
	}

	// DEBUG: Log response body
	respBytes, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(respBytes))
	log.Debug("erp response body", slog.String("body", logger.Body(string(respBytes))))

	latency := int(time.Since(start).Milliseconds())
	if resp.StatusCode >= 400 {
		log.Warn("erp non-2xx response",
			slog.String("path", ep.Path),
			slog.Int("status_code", resp.StatusCode),
			slog.Int("latency_ms", latency),
		)
	} else {
		log.Info("erp response",
			slog.String("path", ep.Path),
			slog.Int("status_code", resp.StatusCode),
			slog.Int("latency_ms", latency),
		)
	}

	return resp, nil
}
