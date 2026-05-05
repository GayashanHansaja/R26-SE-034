package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/cache"
)

type Server struct {
	mcpServer *server.MCPServer
	connector ERPConnector
	cache     *cache.Manager
	tools     map[string]*Tool // Internal storage for our Tool metadata
}

func NewServer(connector ERPConnector, cacheMgr *cache.Manager) *Server {
	s := server.NewMCPServer("ERPBridge", "1.0.0")
	return &Server{
		mcpServer: s,
		connector: connector,
		cache:     cacheMgr,
		tools:     make(map[string]*Tool),
	}
}

// RegisterTool adds a tool to the server's registry at runtime.
func (s *Server) RegisterTool(t *Tool) {
	s.tools[t.Name] = t

	// Serialize the input schema to JSON.RawMessage
	schemaJSON, err := json.Marshal(t.InputSchema)
	if err != nil {
		log.Printf("failed to marshal input schema for %s: %v", t.Name, err)
		return
	}

	// Create mcp-go tool using RawInputSchema
	mcpTool := mcp.NewTool(t.Name, 
		mcp.WithDescription(t.Description),
		mcp.WithRawInputSchema(json.RawMessage(schemaJSON)),
	)

	// Add tool to server with handler
	s.mcpServer.AddTool(mcpTool, s.handleMCPToolCall(t.Name))
	log.Printf("registered MCP tool: %s", t.Name)
}

func (s *Server) handleMCPToolCall(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t, ok := s.tools[name]
		if !ok {
			return nil, fmt.Errorf("tool not found: %s", name)
		}

		// Type assertion for arguments
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok && request.Params.Arguments != nil {
			return nil, fmt.Errorf("invalid arguments format")
		}

		// Caching Layer — READ
		var role string // TODO: extract from context if Phase 3 RBAC is present
		if s.cache != nil && t.Cache != nil && t.Cache.Enabled {
			entry, err := s.cache.Get(ctx, t.Name, role, args, *t.Cache)
			if err == nil && entry != nil && entry.HitType != "miss" {
				log.Printf("cache hit [%s] for %s", entry.HitType, t.Name)
				return mcp.NewToolResultText(string(entry.Response)), nil
			}
		}

		result, err := t.Execute(ctx, args, s.connector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Caching Layer — WRITE (Store)
		if s.cache != nil && t.Cache != nil && t.Cache.Enabled && !result.IsError {
			respJSON, _ := json.Marshal(result.Result)
			if err := s.cache.Set(ctx, t.Name, role, args, respJSON, *t.Cache); err != nil {
				log.Printf("warn: failed to cache result for %s: %v", t.Name, err)
			}
		}

		// Invalidation Layer — AUTO-FLUSH
		if s.cache != nil && t.Cache != nil && len(t.Cache.FlushOn) > 0 && !result.IsError {
			if err := s.cache.AutoFlush(ctx, t.Cache.FlushOn); err != nil {
				log.Printf("warn: auto-flush failed for %s: %v", t.Name, err)
			}
		}

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

	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	t, ok := s.tools[req.Name]
	if !ok {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}

	// Caching Layer — READ
	var role string // TODO: extract from headers if present
	if s.cache != nil && t.Cache != nil && t.Cache.Enabled {
		entry, err := s.cache.Get(r.Context(), t.Name, role, req.Arguments, *t.Cache)
		if err == nil && entry != nil && entry.HitType != "miss" {
			log.Printf("cache hit [%s] for %s (direct)", entry.HitType, t.Name)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache-Hit", entry.HitType)
			json.NewEncoder(w).Encode(ToolResult{Result: entry.Response})
			return
		}
	}

	result, err := t.Execute(r.Context(), req.Arguments, s.connector)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Caching Layer — WRITE (Store)
	if s.cache != nil && t.Cache != nil && t.Cache.Enabled && !result.IsError {
		respJSON, _ := json.Marshal(result.Result)
		if err := s.cache.Set(r.Context(), t.Name, role, req.Arguments, respJSON, *t.Cache); err != nil {
			log.Printf("warn: failed to cache result for %s: %v", t.Name, err)
		}
	}

	// Invalidation Layer — AUTO-FLUSH
	if s.cache != nil && t.Cache != nil && len(t.Cache.FlushOn) > 0 && !result.IsError {
		if err := s.cache.AutoFlush(r.Context(), t.Cache.FlushOn); err != nil {
			log.Printf("warn: auto-flush failed for %s: %v", t.Name, err)
		}
	}

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
