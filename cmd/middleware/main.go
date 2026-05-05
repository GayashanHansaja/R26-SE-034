package main

import (
	"encoding/json"
	"io/ioutil"
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

	conn := connector.NewClient()
	server := mcp.NewServer(conn)

	// Load tools from schemas directory
	loadTools(server, schemasDir)

	mux := http.NewServeMux()
	server.ServeHTTP(mux)

	log.Printf("Bridge Middleware listening on :%s", mcpPort)
	log.Fatal(http.ListenAndServe(":"+mcpPort, mux))
}

func loadTools(s *mcp.Server, dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			data, err := ioutil.ReadFile(path)
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
