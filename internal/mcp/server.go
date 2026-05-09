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
	"strings"
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
	store           *Store
	registry        *ToolRegistry
	resources       map[string]*Resource
	prompts         map[string]*Prompt
	Notifier        *CustomNotifier
	toolMiddlewares []server.ToolHandlerMiddleware
}

// RateLimitConfig defines the configuration for the tool rate limiter.
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// NewServer creates a new Server instance with the provided connector, cache manager, and logger.
func NewServer(connector ERPConnector, cacheMgr *cache.Manager, rootLog *slog.Logger, rateCfg RateLimitConfig, dbPath string) *Server {
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

	store, err := NewStore(dbPath)
	if err != nil {
		mcpLog.Error("failed to initialize store", slog.String("error", err.Error()))
	}

	srv := &Server{
		mcpServer: s,
		connector: connector,
		cache:     cacheMgr,
		log:       mcpLog,
		store:     store,
		registry:  NewToolRegistry(),
		resources: make(map[string]*Resource),
		prompts:   make(map[string]*Prompt),
		Notifier:  NewCustomNotifier(s),
	}

	// Initialize global tool middlewares
	rateLimiter := NewRateLimitMiddleware(rateCfg.RequestsPerSecond, rateCfg.Burst)
	srv.toolMiddlewares = []server.ToolHandlerMiddleware{
		rateLimiter.Handle(),
		LoggingMiddleware(srv.log),
		MetricsMiddleware(),
	}

	srv.RegisterBuiltinTools()

	// Start Reconciliation Loop
	go srv.StartController(context.Background())

	return srv
}

// RegisterBuiltinTools registers internal system tools.
func (s *Server) RegisterBuiltinTools() {
	s.RegisterTool(&Tool{
		APIVersion: "erpbridge.io/v1",
		Kind:       "MCPTool",
		Metadata: Metadata{
			Name:    "system.progress_test",
			Version: "1.0.0",
			Module:  "system",
		},
		Spec: ToolSpec{
			Description: Description{
				Short: "A demonstration tool that sends real-time progress notifications.",
			},
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
		APIVersion: "erpbridge.io/v1",
		Kind:       "MCPTool",
		Metadata: Metadata{
			Name:    "system.sensitive_log_test",
			Version: "1.0.0",
			Module:  "system",
		},
		Spec: ToolSpec{
			Description: Description{
				Short: "A demonstration tool that logs sensitive data to verify redaction.",
			},
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
		},
		Handler: func(ctx context.Context, args map[string]any) (*ToolResult, error) {
			token, _ := args["token"].(string)
			msg, _ := args["message"].(string)

			// Log using the composite logger which includes MCPHandler
			s.log.InfoContext(ctx, "Sensitive data received",
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

// StartController runs the background reconciliation loop.
func (s *Server) StartController(ctx context.Context) {
	s.log.Info("starting reconciliation controller")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Reconcile(ctx)
		}
	}
}

// Reconcile ensures the MCP runtime matches the desired state in the SQLite store.
func (s *Server) Reconcile(ctx context.Context) {
	if s.store == nil {
		return
	}

	desiredTools, err := s.store.List()
	if err != nil {
		s.log.Error("failed to list desired tools", slog.String("error", err.Error()))
		return
	}

	// Simple reconciliation: Register any tool that is in the store but not in registry,
	// or has a different version/spec.
	for _, dt := range desiredTools {
		existing, err := s.registry.Resolve(dt.Metadata.Name, dt.Metadata.Version)
		if err != nil || existing == nil {
			s.log.Info("reconciling tool", slog.String("name", dt.Metadata.Name), slog.String("version", dt.Metadata.Version))
			s.RegisterTool(dt)
		}
	}
}

// RegisterTool adds a tool to the server's registry and active MCP server.
func (s *Server) RegisterTool(t *Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.registry.Add(t); err != nil {
		s.log.Error("failed to add tool to registry", slog.String("tool", t.Metadata.Name), slog.String("error", err.Error()))
		return
	}

	// Serialize the input schema to JSON.RawMessage
	schemaJSON, err := json.Marshal(t.Spec.InputSchema)
	if err != nil {
		s.log.Error("failed to marshal input schema", slog.String("tool_name", t.Metadata.Name), slog.String("error", err.Error()))
		return
	}

	// Use versioned name internally for MCP registration if it's not the default?
	// Actually, the spec says we should use stable aliases.
	// For now, we register with the base name and let the handler resolve.
	mcpTool := mcp.NewToolWithRawSchema(t.Metadata.Name, t.Spec.Description.Short, json.RawMessage(schemaJSON))

	// Explicitly clear structured schema fields to avoid conflict during marshaling
	mcpTool.InputSchema = mcp.ToolInputSchema{}
	mcpTool.OutputSchema = mcp.ToolOutputSchema{}

	// Add tool to server with handler
	handler := s.handleMCPToolCall(t.Metadata.Name)

	// Apply tool-specific middlewares
	handler = s.CacheMiddleware(t)(handler)

	// Apply global middlewares
	for i := len(s.toolMiddlewares) - 1; i >= 0; i-- {
		handler = s.toolMiddlewares[i](handler)
	}

	s.mcpServer.AddTool(mcpTool, handler)
	s.log.Info("registered MCP tool", slog.String("tool_name", t.Metadata.Name), slog.String("version", t.Metadata.Version))

	// Notify clients that tools have changed
	s.mcpServer.SendNotificationToAllClients("notifications/tools/list_changed", nil)
}

func (s *Server) handleMCPToolCall(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		s.mu.RLock()
		// Resolve the latest stable version for this tool name
		t, err := s.registry.Resolve(name, "")
		s.mu.RUnlock()

		if err != nil {
			return nil, fmt.Errorf("tool not found: %s (%w)", name, err)
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

	// 4. Kubernetes-Style Tool API
	mux.HandleFunc("/apis/erpbridge.io/v1/tools", s.handleToolAPI)
}

func (s *Server) handleToolAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleToolApply(w, r)
	case http.MethodGet:
		s.handleToolList(w, r)
	case http.MethodDelete:
		s.handleToolDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleToolApply(w http.ResponseWriter, r *http.Request) {
	var t Tool
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Admission Controller
	if err := s.validateTool(&t); err != nil {
		http.Error(w, "invalid tool: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if s.store == nil {
		http.Error(w, "store not available", http.StatusInternalServerError)
		return
	}

	if err := s.store.Save(&t); err != nil {
		http.Error(w, "failed to save tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Immediate reconciliation for responsiveness
	s.RegisterTool(&t)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "applied",
		"name":    t.Metadata.Name,
		"version": t.Metadata.Version,
	})
}

func (s *Server) handleToolList(w http.ResponseWriter, r *http.Request) {
	tools, err := s.store.List()
	if err != nil {
		http.Error(w, "failed to list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tools)
}

func (s *Server) handleToolDelete(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	version := r.URL.Query().Get("version")

	if name == "" || version == "" {
		http.Error(w, "missing name or version parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.Delete(name, version); err != nil {
		http.Error(w, "failed to delete tool: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Admission Controller
func (s *Server) validateTool(t *Tool) error {
	if t.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if t.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	if strings.Contains(strings.ToLower(t.Metadata.Name), "get-") ||
		strings.Contains(strings.ToLower(t.Metadata.Name), "post-") {
		return fmt.Errorf("tool name should be intent-based, not include HTTP verbs")
	}

	// Check for embedded secrets in Execution path (simplified check)
	if strings.Contains(t.Spec.Execution.Endpoint, "token ") ||
		strings.Contains(t.Spec.Execution.Endpoint, "key=") {
		return fmt.Errorf("endpoint should not contain raw secrets, use credentialRef instead")
	}

	return nil
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
	// Resolve tool by name (latest stable)
	t, err := s.registry.Resolve(req.Name, "")
	s.mu.RUnlock()

	if err != nil {
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
