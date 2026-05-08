// Package mcp implements the Model Context Protocol (MCP) server,
// allowing AI agents to interact with ERP systems through tools,
// resources, and prompts.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/logger"
)

// Server is the primary MCP server implementation for ERPBridge.
type Server struct {
	mcpServer       *server.MCPServer
	connector       ERPConnector
	cache           *cache.Manager
	log             *slog.Logger
	mu              sync.RWMutex
	tools           map[string]*Tool
	resources       map[string]*Resource
	prompts         map[string]*Prompt
	Notifier        *CustomNotifier
	toolMiddlewares []server.ToolHandlerMiddleware
}

// NewServer creates a new Server instance with the provided connector, cache manager, and logger.
func NewServer(connector ERPConnector, cacheMgr *cache.Manager, rootLog *slog.Logger) *Server {
	s := server.NewMCPServer("ERPBridge", "1.0.0",
		server.WithLogging(),
		server.WithResourceCompletionProvider(&ResourceCompletionProvider{}),
		server.WithPromptCompletionProvider(&PromptCompletionProvider{}),
	)

	mcpHandler := logger.NewMCPHandler(s, "mcp")
	mcpLog := slog.New(logger.MultiHandler{
		logger.Component(rootLog, "mcp").Handler(),
		mcpHandler,
	})

	srv := &Server{
		mcpServer: s,
		connector: connector,
		cache:     cacheMgr,
		log:       mcpLog,
		tools:     make(map[string]*Tool),
		resources: make(map[string]*Resource),
		prompts:   make(map[string]*Prompt),
		Notifier:  NewCustomNotifier(s),
	}

	// Initialize global tool middlewares
	srv.toolMiddlewares = []server.ToolHandlerMiddleware{
		LoggingMiddleware(srv.log),
		MetricsMiddleware(),
	}

	srv.RegisterBuiltinTools()

	return srv
}

// RegisterBuiltinTools registers internal system tools.
func (s *Server) RegisterBuiltinTools() {
	s.RegisterTool(&Tool{
		Name:        "system.progress_test",
		Description: "A demonstration tool that sends real-time progress notifications.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"steps": {
					Type:        "integer",
					Description: "Number of steps to simulate (max 100).",
					Default:     10,
				},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (*ToolResult, error) {
			steps := 10
			if s, ok := args["steps"].(float64); ok {
				steps = int(s)
			}
			if steps > 100 {
				steps = 100
			}

			for i := 1; i <= steps; i++ {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(200 * time.Millisecond):
					s.Notifier.SendProgress(ctx, i, steps, fmt.Sprintf("Processing step %d/%d...", i, steps))
				}
			}

			return &ToolResult{
				Result: map[string]any{
					"status":  "completed",
					"message": fmt.Sprintf("Finished %d steps successfully.", steps),
				},
			}, nil
		},
	})

	s.RegisterTool(&Tool{
		Name:        "system.sensitive_log_test",
		Description: "A demonstration tool that logs sensitive data to verify redaction.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"token": {
					Type:        "string",
					Description: "A sensitive token that should be redacted.",
				},
				"message": {
					Type:        "string",
					Description: "A normal message.",
				},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (*ToolResult, error) {
			token, _ := args["token"].(string)
			msg, _ := args["message"].(string)

			// Log using the composite logger which includes MCPHandler
			s.log.Info("Sensitive data received",
				slog.String("token", token), // Should be redacted by field name
				slog.String("message", msg), // Should be preserved
				slog.Any("raw_args", args),  // Should be redacted by keys in map
			)

			return &ToolResult{
				Result: map[string]any{
					"status":  "success",
					"message": "Logs emitted. Check your MCP client logs.",
				},
			}, nil
		},
	})
}

// ResourceCompletionProvider implements the mcp-go completion provider for resources.
type ResourceCompletionProvider struct{}

// CompleteResourceArgument provides suggestions for resource URIs.
func (p *ResourceCompletionProvider) CompleteResourceArgument(ctx context.Context, uri string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error) {
	return &mcp.Completion{
		Values: []string{"recent-item-1", "recent-item-2"},
	}, nil
}

// PromptCompletionProvider implements the mcp-go completion provider for prompts.
type PromptCompletionProvider struct{}

// CompletePromptArgument provides suggestions for prompt arguments.
func (p *PromptCompletionProvider) CompletePromptArgument(ctx context.Context, name string, argument mcp.CompleteArgument, context mcp.CompleteContext) (*mcp.Completion, error) {
	return &mcp.Completion{
		Values: []string{"suggested-value-A", "suggested-value-B"},
	}, nil
}

// RegisterResource adds a resource to the server.
func (s *Server) RegisterResource(r *Resource) {
	s.mu.Lock()
	s.resources[r.URITemplate] = r
	s.mu.Unlock()

	mcpResource := mcp.NewResource(r.URITemplate, r.Name,
		mcp.WithResourceDescription(r.Description),
		mcp.WithMIMEType(r.MimeType),
	)
	s.mcpServer.AddResource(mcpResource, s.handleMCPResourceRead)
	s.log.Info("registered MCP resource", slog.String("name", r.Name), slog.String("uri", r.URITemplate))
}

// RegisterPrompt adds a prompt template to the server.
func (s *Server) RegisterPrompt(p *Prompt) {
	s.mu.Lock()
	s.prompts[p.Name] = p
	s.mu.Unlock()
	mcpPrompt := mcp.NewPrompt(p.Name,
		mcp.WithPromptDescription(p.Description),
	)
	for _, a := range p.Arguments {
		mcpPrompt.Arguments = append(mcpPrompt.Arguments, mcp.PromptArgument{
			Name:        a.Name,
			Description: a.Description,
			Required:    a.Required,
		})
	}
	s.mcpServer.AddPrompt(mcpPrompt, s.handleMCPPromptGet)
	s.log.Info("registered MCP prompt", slog.String("name", p.Name))
}

func (s *Server) handleMCPResourceRead(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Find resource by URI matching (simplistic for template)
	// In a real implementation, we'd use a regex or template matcher
	s.mu.RLock()
	r, ok := s.resources[request.Params.URI]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("resource not found: %s", request.Params.URI)
	}

	content, err := r.Execute(ctx, request.Params.URI, s.connector)
	if err != nil {
		return nil, err
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: r.MimeType,
			Text:     content,
		},
	}, nil
}

func (s *Server) handleMCPPromptGet(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	s.mu.RLock()
	p, ok := s.prompts[request.Params.Name]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", request.Params.Name)
	}

	// Simple template expansion (naive)
	text := p.Template
	if request.Params.Arguments != nil {
		for k, v := range request.Params.Arguments {
			text = fmt.Sprintf("%s\n\n%s: %v", text, k, v)
		}
	}

	return &mcp.GetPromptResult{
		Description: p.Description,
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: text,
				},
			},
		},
	}, nil
}

// RegisterTool adds a tool to the server's registry at runtime.
func (s *Server) RegisterTool(t *Tool) {
	s.mu.Lock()
	s.tools[t.Name] = t
	s.mu.Unlock()

	// Serialize the input schema to JSON.RawMessage
	schemaJSON, err := json.Marshal(t.InputSchema)
	if err != nil {
		s.log.Error("failed to marshal input schema", slog.String("tool_name", t.Name), slog.String("error", err.Error()))
		return
	}

	// Create mcp-go tool using RawInputSchema directly to avoid conflict with NewTool's default InputSchema
	mcpTool := mcp.NewToolWithRawSchema(t.Name, t.Description, json.RawMessage(schemaJSON))

	// Explicitly clear structured schema fields to avoid conflict during marshaling
	mcpTool.InputSchema = mcp.ToolInputSchema{}
	mcpTool.OutputSchema = mcp.ToolOutputSchema{}

	// Add tool to server with handler
	handler := s.handleMCPToolCall(t.Name)

	// Apply tool-specific middlewares
	handler = s.CacheMiddleware(t)(handler)

	// Apply global middlewares
	for i := len(s.toolMiddlewares) - 1; i >= 0; i-- {
		handler = s.toolMiddlewares[i](handler)
	}

	s.mcpServer.AddTool(mcpTool, handler)
	s.log.Info("registered MCP tool", slog.String("tool_name", t.Name))

	// Notify clients that tools have changed
	s.mcpServer.SendNotificationToAllClients("notifications/tools/list_changed", nil)
}

func (s *Server) handleMCPToolCall(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.mu.RLock()
		t, ok := s.tools[name]
		s.mu.RUnlock()

		if !ok {
			return nil, fmt.Errorf("tool not found: %s", name)
		}

		// Type assertion for arguments
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok && request.Params.Arguments != nil {
			return nil, fmt.Errorf("invalid arguments format")
		}

		result, err := t.Execute(ctx, args, s.connector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Convert result to mcp-go format
		resultJSON, _ := json.Marshal(result.Result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// MCPServer returns the underlying mcp-go MCPServer instance.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// ServeHTTP handles the various MCP transports and management endpoints.
func (s *Server) ServeHTTP(mux *http.ServeMux, baseURL string) {
	// 1. Streamable HTTP Transport (Modern clients, Postman)
	// MUST strip prefix so the server sees "/" internally
	streamableServer := server.NewStreamableHTTPServer(s.mcpServer,
		server.WithStateful(true),
		server.WithSessionIdleTTL(30*time.Minute),
		server.WithEndpointPath("/"), // Tell the server it is mounted at /
		server.WithStreamableHTTPCORS(
			server.WithCORSAllowedOrigins("*"),
			server.WithCORSAllowedMethods("POST", "GET", "OPTIONS"),
			server.WithCORSAllowedHeaders("Content-Type", "Mcp-Session-Id", "Last-Event-ID", "Authorization"),
			server.WithCORSExposedHeaders("Mcp-Session-Id"),
		),
	)
	mux.Handle("/mcp/", http.StripPrefix("/mcp", streamableServer))

	// 3. Management & Utility Endpoints
	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/api/tools/invoke", s.handleDirectInvoke)
	mux.HandleFunc("/api/cache/stats", s.handleCacheStats)
	mux.HandleFunc("/api/cache/flush", s.handleCacheFlush)
	mux.HandleFunc("/api/cache/list", s.handleCacheList)
	mux.HandleFunc("/api/cache/inspect", s.handleCacheInspect)
	mux.HandleFunc("/api/logs/stream", s.handleLogStream)
	mux.HandleFunc("/api/logs/recent", s.handleLogRecent)
}

func (s *Server) handleDirectInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error("bad request", slog.String("error", err.Error()))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	t, ok := s.tools[req.Name]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	// Create a bridge to the MCP middleware chain
	mcpReq := mcp.CallToolRequest{}
	mcpReq.Params.Name = req.Name
	mcpReq.Params.Arguments = req.Arguments

	// Base handler for direct invoke
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]any)
		result, err := t.Execute(ctx, args, s.connector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resultJSON, _ := json.Marshal(result.Result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	// Apply tool-specific middlewares
	handler = s.CacheMiddleware(t)(handler)

	// Apply global middlewares
	for i := len(s.toolMiddlewares) - 1; i >= 0; i-- {
		handler = s.toolMiddlewares[i](handler)
	}

	// Execute through the middleware chain
	mcpResult, err := handler(r.Context(), mcpReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if mcpResult.IsError {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": mcpResult.Content})
		return
	}

	// Convert back to internal ToolResult for compatibility if needed,
	// or just send the MCP result content.
	// For bridgectl compatibility, we send ToolResult structure.
	var result any
	if len(mcpResult.Content) > 0 {
		if text, ok := mcpResult.Content[0].(mcp.TextContent); ok {
			_ = json.Unmarshal([]byte(text.Text), &result)
		}
	}

	_ = json.NewEncoder(w).Encode(ToolResult{Result: result})
}

func (s *Server) handleCacheFlush(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		http.Error(w, "cache not enabled", http.StatusServiceUnavailable)
		return
	}

	tool := r.URL.Query().Get("tool")
	module := r.URL.Query().Get("module")
	all := r.URL.Query().Get("all") == "true"

	var count int
	var err error

	if all {
		count, err = s.cache.FlushModule(r.Context(), "") // Empty matches all exact
	} else if tool != "" {
		count, err = s.cache.FlushTool(r.Context(), tool)
	} else if module != "" {
		count, err = s.cache.FlushModule(r.Context(), module)
	} else {
		http.Error(w, "missing tool, module or all parameter", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"deleted": count,
		"status":  "ok",
	})
}

func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		http.Error(w, "cache not enabled", http.StatusServiceUnavailable)
		return
	}

	stats, err := s.cache.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apiVersion": "v1",
		"kind":       "CacheStats",
		"status":     "active",
		"stats":      stats,
	})
}

func (s *Server) handleCacheList(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleCacheInspect(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := logger.Subscribe()
	defer logger.Unsubscribe(ch)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(msg))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleLogRecent(w http.ResponseWriter, r *http.Request) {
	logs := logger.GetRecentLogs()
	w.Header().Set("Content-Type", "application/json")

	// Logs are already JSON strings
	_, _ = fmt.Fprintf(w, "[")
	for i, l := range logs {
		if i > 0 {
			_, _ = fmt.Fprintf(w, ",")
		}
		_, _ = fmt.Fprintf(w, "%s", string(l))
	}
	_, _ = fmt.Fprintf(w, "]")
}
