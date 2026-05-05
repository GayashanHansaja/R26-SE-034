package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/logger"
)

type Server struct {
	mcpServer *server.MCPServer
	connector ERPConnector
	cache     *cache.Manager
	log       *slog.Logger
	tools     map[string]*Tool // Internal storage for our Tool metadata
}

func NewServer(connector ERPConnector, cacheMgr *cache.Manager, rootLog *slog.Logger) *Server {
	s := server.NewMCPServer("ERPBridge", "1.0.0")
	return &Server{
		mcpServer: s,
		connector: connector,
		cache:     cacheMgr,
		log:       logger.Component(rootLog, "mcp"),
		tools:     make(map[string]*Tool),
	}
}

// RegisterTool adds a tool to the server's registry at runtime.
func (s *Server) RegisterTool(t *Tool) {
	s.tools[t.Name] = t

	// Serialize the input schema to JSON.RawMessage
	schemaJSON, err := json.Marshal(t.InputSchema)
	if err != nil {
		s.log.Error("failed to marshal input schema", slog.String("tool_name", t.Name), slog.String("error", err.Error()))
		return
	}

	// Create mcp-go tool using RawInputSchema
	mcpTool := mcp.NewTool(t.Name, 
		mcp.WithDescription(t.Description),
		mcp.WithRawInputSchema(json.RawMessage(schemaJSON)),
	)

	// Add tool to server with handler
	s.mcpServer.AddTool(mcpTool, s.handleMCPToolCall(t.Name))
	s.log.Info("registered MCP tool", slog.String("tool_name", t.Name))
}

func (s *Server) handleMCPToolCall(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		reqID := logger.NewRequestID()

		t, ok := s.tools[name]
		if !ok {
			s.log.Warn("tool not found", slog.String("request_id", reqID), slog.String("tool_name", name))
			return nil, fmt.Errorf("tool not found: %s", name)
		}

		// Type assertion for arguments
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok && request.Params.Arguments != nil {
			s.log.Error("invalid arguments format", slog.String("request_id", reqID), slog.String("tool_name", name))
			return nil, fmt.Errorf("invalid arguments format")
		}

		// Attach standard fields to the logger for this request
		var role string // Extract from context if available
		reqLog := s.log.With(
			slog.String("request_id", reqID),
			slog.String("tool_name", t.Name),
			slog.String("role", role),
		)
		ctx = logger.WithLogger(ctx, reqLog)

		reqLog.Info("tool call received", slog.Any("arg_keys", logger.ArgKeys(args)))
		reqLog.Debug("tool call arguments", slog.Any("args", logger.Arguments(args)))

		// Caching Layer — READ
		var cacheStatus = "MISS"
		var cacheType = "none"
		if s.cache != nil && t.Cache != nil && t.Cache.Enabled {
			entry, err := s.cache.Get(ctx, t.Name, role, args, *t.Cache)
			if err == nil && entry != nil && entry.HitType != "miss" {
				cacheStatus = "HIT"
				cacheType = entry.HitType
				reqLog.Info("tool call complete",
					slog.Int("latency_ms", int(time.Since(start).Milliseconds())),
					slog.String("cache_status", cacheStatus),
					slog.String("cache_type", cacheType),
				)
				return mcp.NewToolResultText(string(entry.Response)), nil
			}
		}

		result, err := t.Execute(ctx, args, s.connector)
		if err != nil {
			reqLog.Error("tool call failed", slog.String("error", err.Error()), slog.Int("latency_ms", int(time.Since(start).Milliseconds())))
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Caching Layer — WRITE (Store)
		if s.cache != nil && t.Cache != nil && t.Cache.Enabled && !result.IsError {
			respJSON, _ := json.Marshal(result.Result)
			if err := s.cache.Set(ctx, t.Name, role, args, respJSON, *t.Cache); err != nil {
				reqLog.Warn("failed to cache result", slog.String("error", err.Error()))
			}
		}

		// Invalidation Layer — AUTO-FLUSH
		if s.cache != nil && t.Cache != nil && len(t.Cache.FlushOn) > 0 && !result.IsError {
			if err := s.cache.AutoFlush(ctx, t.Cache.FlushOn); err != nil {
				reqLog.Warn("auto-flush failed", slog.String("error", err.Error()))
			}
		}

		reqLog.Info("tool call complete",
			slog.Int("latency_ms", int(time.Since(start).Milliseconds())),
			slog.String("cache_status", cacheStatus),
			slog.String("cache_type", cacheType),
		)

		// Convert result to mcp-go format
		resultJSON, _ := json.Marshal(result.Result)
		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ServeHTTP handles the SSE transport for mcp-go and a direct invocation endpoint for the CLI.
func (s *Server) ServeHTTP(mux *http.ServeMux, baseURL string) {
	// SSE transport for AI agents
	sseServer := server.NewSSEServer(s.mcpServer, server.WithBaseURL(baseURL))
	mux.Handle("/mcp/sse", sseServer.SSEHandler())
	mux.Handle("/mcp/messages", sseServer.MessageHandler())

	// Direct invocation endpoint for bridgectl
	mux.HandleFunc("/api/tools/invoke", s.handleDirectInvoke)

	// Cache management endpoints
	mux.HandleFunc("/api/cache/stats", s.handleCacheStats)
	mux.HandleFunc("/api/cache/flush", s.handleCacheFlush)
	mux.HandleFunc("/api/cache/list", s.handleCacheList)
	mux.HandleFunc("/api/cache/inspect", s.handleCacheInspect)

	// Log streaming endpoint
	mux.HandleFunc("/api/logs/stream", s.handleLogStream)
	mux.HandleFunc("/api/logs/recent", s.handleLogRecent)

	mux.HandleFunc("/mcp/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
}

func (s *Server) handleDirectInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	reqID := logger.NewRequestID()

	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error("bad request", slog.String("request_id", reqID), slog.String("error", err.Error()))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	t, ok := s.tools[req.Name]
	if !ok {
		s.log.Warn("tool not found", slog.String("request_id", reqID), slog.String("tool_name", req.Name))
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	// Attach standard fields to the logger for this request
	var role string // Extract from headers if present
	reqLog := s.log.With(
		slog.String("request_id", reqID),
		slog.String("tool_name", t.Name),
		slog.String("role", role),
	)
	ctx := logger.WithLogger(r.Context(), reqLog)

	reqLog.Info("tool call received (direct)", slog.Any("arg_keys", logger.ArgKeys(req.Arguments)))
	reqLog.Debug("tool call arguments (direct)", slog.Any("args", logger.Arguments(req.Arguments)))

	// Caching Layer — READ
	var cacheStatus = "MISS"
	var cacheType = "none"
	if s.cache != nil && t.Cache != nil && t.Cache.Enabled {
		entry, err := s.cache.Get(ctx, t.Name, role, req.Arguments, *t.Cache)
		if err == nil && entry != nil && entry.HitType != "miss" {
			cacheStatus = "HIT"
			cacheType = entry.HitType
			reqLog.Info("tool call complete (direct)",
				slog.Int("latency_ms", int(time.Since(start).Milliseconds())),
				slog.String("cache_status", cacheStatus),
				slog.String("cache_type", cacheType),
			)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache-Hit", entry.HitType)
			json.NewEncoder(w).Encode(ToolResult{Result: entry.Response})
			return
		}
	}

	result, err := t.Execute(ctx, req.Arguments, s.connector)
	if err != nil {
		reqLog.Error("tool call failed (direct)", slog.String("error", err.Error()), slog.Int("latency_ms", int(time.Since(start).Milliseconds())))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Caching Layer — WRITE (Store)
	if s.cache != nil && t.Cache != nil && t.Cache.Enabled && !result.IsError {
		respJSON, _ := json.Marshal(result.Result)
		if err := s.cache.Set(ctx, t.Name, role, req.Arguments, respJSON, *t.Cache); err != nil {
			reqLog.Warn("failed to cache result (direct)", slog.String("error", err.Error()))
		}
	}

	// Invalidation Layer — AUTO-FLUSH
	if s.cache != nil && t.Cache != nil && len(t.Cache.FlushOn) > 0 && !result.IsError {
		if err := s.cache.AutoFlush(ctx, t.Cache.FlushOn); err != nil {
			reqLog.Warn("auto-flush failed (direct)", slog.String("error", err.Error()))
		}
	}

	reqLog.Info("tool call complete (direct)",
		slog.Int("latency_ms", int(time.Since(start).Milliseconds())),
		slog.String("cache_status", cacheStatus),
		slog.String("cache_type", cacheType),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
	json.NewEncoder(w).Encode(map[string]any{
		"deleted": count,
		"status":  "ok",
	})
}

func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		http.Error(w, "cache not enabled", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"apiVersion": "v1",
		"kind":       "CacheStats",
		"summary": map[string]any{
			"status": "active",
		},
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
			fmt.Fprintf(w, "data: %s\n\n", string(msg))
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
	fmt.Fprintf(w, "[")
	for i, l := range logs {
		if i > 0 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, "%s", string(l))
	}
	fmt.Fprintf(w, "]")
}
