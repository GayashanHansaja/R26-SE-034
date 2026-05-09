package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestServer_RegisterResource(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	r := &Resource{
		Name:        "test-resource",
		Description: "Test",
		URITemplate: "erp://test",
		MimeType:    "text/plain",
	}

	s.RegisterResource(r)

	assert.NotNil(t, s.resources["erp://test"])
	assert.Equal(t, "test-resource", s.resources["erp://test"].Name)
}

func TestServer_HandleResourceRead(t *testing.T) {
	log := logger.Init()

	mockConn := &MockConnector{
		CallFunc: func(ctx context.Context, ep connector.EndpointConfig, queryParams url.Values, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString("resource content")),
			}, nil
		},
	}
	s := NewServer(mockConn, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	r := &Resource{
		Name:        "test-resource",
		URITemplate: "erp://test",
		MimeType:    "text/plain",
		Execution:   Execution{Endpoint: "/test"},
	}
	s.RegisterResource(r)

	req := mcp.ReadResourceRequest{}
	req.Params.URI = "erp://test"

	res, err := s.handleMCPResourceRead(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	textRes := res[0].(mcp.TextResourceContents)
	assert.Equal(t, "resource content", textRes.Text)

	// test not found
	req.Params.URI = "erp://unknown"
	_, err = s.handleMCPResourceRead(context.Background(), req)
	assert.Error(t, err)
}

func TestServer_RegisterPrompt(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	p := &Prompt{
		Name:        "test-prompt",
		Description: "Test prompt",
		Template:    "Do something",
		Arguments: []PromptArgument{
			{Name: "arg1", Description: "Arg 1", Required: true},
		},
	}

	s.RegisterPrompt(p)

	assert.NotNil(t, s.prompts["test-prompt"])
	assert.Equal(t, "test-prompt", s.prompts["test-prompt"].Name)
}

func TestServer_HandlePromptGet(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	p := &Prompt{
		Name:     "test-prompt",
		Template: "Hello {{name}}",
	}
	s.RegisterPrompt(p)

	req := mcp.GetPromptRequest{}
	req.Params.Name = "test-prompt"
	req.Params.Arguments = map[string]string{"name": "world"}

	res, err := s.handleMCPPromptGet(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Messages, 1)

	textContent := res.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, textContent, "Hello {{name}}")
	assert.Contains(t, textContent, "name: world")

	// test not found
	req.Params.Name = "unknown-prompt"
	_, err = s.handleMCPPromptGet(context.Background(), req)
	assert.Error(t, err)
}

func TestServer_Completions(t *testing.T) {
	rp := &ResourceCompletionProvider{}
	resComp, _ := rp.CompleteResourceArgument(context.Background(), "", mcp.CompleteArgument{}, mcp.CompleteContext{})
	assert.NotNil(t, resComp)
	assert.NotEmpty(t, resComp.Values)

	pp := &PromptCompletionProvider{}
	pComp, _ := pp.CompletePromptArgument(context.Background(), "", mcp.CompleteArgument{}, mcp.CompleteContext{})
	assert.NotNil(t, pComp)
	assert.NotEmpty(t, pComp.Values)
}

func TestServer_RegisterToolMarshaling(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	schema := InputSchema{
		Type: "object",
		Properties: map[string]Property{
			"test": {Type: "string"},
		},
	}

	s.RegisterTool(&Tool{
		Metadata: Metadata{
			Name:    "test-tool",
			Version: "1.0.0",
		},
		Spec: ToolSpec{
			Description: Description{Short: "Test"},
			InputSchema: schema,
		},
	})

	// Access the registered tool from the registry
	tool, err := s.registry.Resolve("test-tool", "")
	assert.NoError(t, err)
	assert.NotNil(t, tool)

	// Try to marshal the tools list as the server would during a tools/list request
	schemaJSON, _ := json.Marshal(schema)
	mcpTool := mcp.NewToolWithRawSchema("test-tool", "Test", json.RawMessage(schemaJSON))

	// Apply the same fix as in RegisterTool
	mcpTool.InputSchema = mcp.ToolInputSchema{}
	mcpTool.OutputSchema = mcp.ToolOutputSchema{}

	data, err := json.Marshal(mcpTool)
	assert.NoError(t, err, "Tool marshaling should not fail")
	assert.Contains(t, string(data), "\"inputSchema\":{\"type\":\"object\"")
}

func TestServer_ServeHTTP(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")
	mux := http.NewServeMux()
	s.ServeHTTP(mux, "http://localhost:8080")

	// test health endpoint
	req := httptest.NewRequest("GET", "/mcp/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

func TestServer_LogStream(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	req := httptest.NewRequest("GET", "/api/logs/stream", nil)

	// Create a context that will cancel quickly so the stream loop exits
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Trigger a log in a goroutine so it gets captured while streaming
	go func() {
		time.Sleep(10 * time.Millisecond)
		log.Info("stream log msg")
	}()

	s.handleLogStream(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "stream log msg")
}

func TestServer_HttpEndpoints(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")
	assert.NotNil(t, s.MCPServer())

	// Test cache flush (not enabled)
	req := httptest.NewRequest("POST", "/api/cache/flush?all=true", nil)
	w := httptest.NewRecorder()
	s.handleCacheFlush(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Test cache stats (not enabled)
	req = httptest.NewRequest("GET", "/api/cache/stats", nil)
	w = httptest.NewRecorder()
	s.handleCacheStats(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Enable cache with a dummy manager to cover parameter validation
	s.cache = cache.NewManager(nil, log)

	req = httptest.NewRequest("POST", "/api/cache/flush", nil)
	w = httptest.NewRecorder()
	s.handleCacheFlush(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test list, inspect (not implemented)
	req = httptest.NewRequest("GET", "/api/cache/list", nil)
	w = httptest.NewRecorder()
	s.handleCacheList(w, req)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	req = httptest.NewRequest("GET", "/api/cache/inspect", nil)
	w = httptest.NewRecorder()
	s.handleCacheInspect(w, req)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	// Test Log recent
	log.Info("test log msg")
	time.Sleep(100 * time.Millisecond)
	req = httptest.NewRequest("GET", "/api/logs/recent", nil)
	w = httptest.NewRecorder()
	s.handleLogRecent(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test log msg")
}

func TestServer_DirectInvoke(t *testing.T) {
	log := logger.Init()
	s := NewServer(nil, nil, log, RateLimitConfig{RequestsPerSecond: 100, Burst: 100}, ":memory:")

	s.RegisterTool(&Tool{
		Metadata: Metadata{
			Name:    "test-invoke",
			Version: "1.0.0",
		},
		Handler: func(ctx context.Context, args map[string]any) (*ToolResult, error) {
			return &ToolResult{Result: map[string]any{"ok": true}}, nil
		},
	})

	// GET not allowed
	req := httptest.NewRequest("GET", "/api/tools/invoke", nil)
	w := httptest.NewRecorder()
	s.handleDirectInvoke(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	// Valid POST
	body := `{"name":"test-invoke","arguments":{}}`
	req = httptest.NewRequest("POST", "/api/tools/invoke", bytes.NewBufferString(body))
	w = httptest.NewRecorder()
	s.handleDirectInvoke(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok":true`)

	// Tool not found
	bodyNotFound := `{"name":"unknown","arguments":{}}`
	req = httptest.NewRequest("POST", "/api/tools/invoke", bytes.NewBufferString(bodyNotFound))
	w = httptest.NewRecorder()
	s.handleDirectInvoke(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
