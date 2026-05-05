package mcp

import (
	"encoding/json"
	"log"
	"net/http"
)

type Server struct {
	tools     map[string]*Tool
	connector ERPConnector
}

func NewServer(connector ERPConnector) *Server {
	return &Server{
		tools:     make(map[string]*Tool),
		connector: connector,
	}
}

// RegisterTool adds a tool to the server's registry at runtime.
func (s *Server) RegisterTool(t *Tool) {
	s.tools[t.Name] = t
	log.Printf("registered MCP tool: %s", t.Name)
}

// ServeHTTP handles both the MCP tool-list and tool-call endpoints.
func (s *Server) ServeHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/mcp/tools/list",   s.handleToolList)
	mux.HandleFunc("/mcp/tools/call",   s.handleToolCall)
	mux.HandleFunc("/mcp/health",       s.handleHealth)
}

func (s *Server) handleToolList(w http.ResponseWriter, r *http.Request) {
	tools := make([]*Tool, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"tools": tools})
}

func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	var req ToolCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad request"})
		return
	}
	tool, ok := s.tools[req.Name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "tool not found"})
		return
	}
	result, err := tool.Execute(r.Context(), req.Arguments, s.connector)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
