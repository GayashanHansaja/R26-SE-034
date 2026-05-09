package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestServer_RegisterTool(t *testing.T) {
	log := logger.Init()
	mockConn := &MockConnector{}
	s := NewServer(mockConn, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	tool := &Tool{
		Metadata: Metadata{
			Name:    "test-tool",
			Version: "1.0.0",
		},
		Spec: ToolSpec{
			Description: Description{Short: "A test tool"},
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"param1": {Type: "string"},
				},
			},
		},
	}

	s.RegisterTool(tool)

	registered, err := s.registry.Resolve("test-tool", "")
	assert.NoError(t, err)
	assert.NotNil(t, registered)
	assert.Equal(t, "test-tool", registered.Metadata.Name)
}

func TestTool_Execute(t *testing.T) {
	mockConn := &MockConnector{
		CallFunc: func(ctx context.Context, ep connector.EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error) {
			respBody := `{"status": "success"}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(respBody)),
			}, nil
		},
	}

	tool := &Tool{
		Metadata: Metadata{Name: "test-tool"},
		Spec: ToolSpec{
			Execution: Execution{
				Method:   "GET",
				Endpoint: "/test",
			},
		},
	}

	ctx := context.Background()
	args := map[string]any{"key": "value"}
	result, err := tool.Execute(ctx, args, mockConn)

	assert.NoError(t, err)
	assert.False(t, result.IsError)

	resultMap, ok := result.Result.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "success", resultMap["status"])
}

func TestTool_Execute_Error(t *testing.T) {
	mockConn := &MockConnector{
		CallFunc: func(ctx context.Context, ep connector.EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "internal server error"}`)),
			}, nil
		},
	}

	tool := &Tool{
		Metadata: Metadata{Name: "test-tool"},
		Spec: ToolSpec{
			Execution: Execution{
				Method:   "POST",
				Endpoint: "/test",
			},
		},
	}

	ctx := context.Background()
	args := map[string]any{"key": "value"}
	result, err := tool.Execute(ctx, args, mockConn)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}
