package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer *server.MCPServer
	connector ERPConnector
	tools     map[string]*Tool // Internal storage for our Tool metadata
}

func NewServer(connector ERPConnector) *Server {
	s := server.NewMCPServer("ERPBridge", "1.0.0")
	return &Server{
		mcpServer: s,
		connector: connector,
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

		result, err := t.Execute(ctx, args, s.connector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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

	result, err := t.Execute(r.Context(), req.Arguments, s.connector)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
