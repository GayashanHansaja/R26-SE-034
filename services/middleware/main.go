package main

import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "path/filepath"

        "github.com/redis/go-redis/v9"
        "github.com/nimendra/ERPBridge/internal/cache"
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

        redisURL := os.Getenv("REDIS_URL")
        embedderURL := os.Getenv("EMBEDDER_URL")

        // In a real scenario, this should be the public URL of the server
        baseURL := os.Getenv("BASE_URL")
        if baseURL == "" {
                baseURL = fmt.Sprintf("http://localhost:%s", mcpPort)
        }

        // Initialize Cache
        var cacheMgr *cache.Manager
        if redisURL != "" {
                opt, err := redis.ParseURL(redisURL)
                if err != nil {
                        log.Printf("failed to parse redis url: %v", err)
                } else {
                        rdb := redis.NewClient(opt)
                        var embedder cache.Embedder
                        if embedderURL != "" {
                                embedder = cache.NewHFEmbedder(embedderURL)
                        }
                        cacheMgr = cache.NewManager(rdb, embedder)
                        if err := cacheMgr.EnsureIndex(context.Background()); err != nil {
                                log.Printf("warn: failed to ensure redis index: %v", err)
                        }
                        log.Printf("cache initialized with Redis at %s", redisURL)
                }
        }

        conn := connector.NewClient()
        server := mcp.NewServer(conn, cacheMgr)
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
