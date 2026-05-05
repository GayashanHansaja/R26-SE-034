package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/nimendra/ERPBridge/internal/mcp"
)

func main() {
	mcpPort := os.Getenv("MCP_PORT")
	if mcpPort == "" {
		mcpPort = "8080"
	}

	schemasDir := os.Getenv("SCHEMAS_DIR")
	if schemasDir == "" {
		schemasDir = "schemas"
	}

	// In a real scenario, this should be the public URL of the server
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", mcpPort)
	}

	conn := connector.NewClient()
	server := mcp.NewServer(conn)

	// Load tools from schemas directory
	loadTools(server, schemasDir)

	mux := http.NewServeMux()
	server.ServeHTTP(mux, baseURL)

	log.Printf("Bridge Middleware listening on :%s", mcpPort)
	log.Printf("MCP SSE endpoint: %s/mcp/sse", baseURL)
	log.Fatal(http.ListenAndServe(":"+mcpPort, mux))
}

func loadTools(s *mcp.Server, dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			data, err := os.ReadFile(path)
			if err != nil {
				log.Printf("failed to read schema %s: %v", path, err)
				return nil
			}
			var tool mcp.Tool
			if err := json.Unmarshal(data, &tool); err != nil {
				log.Printf("failed to unmarshal schema %s: %v", path, err)
				return nil
			}
			s.RegisterTool(&tool)
		}
		return nil
	})
	if err != nil {
		log.Printf("error walking schemas directory: %v", err)
	}
}
