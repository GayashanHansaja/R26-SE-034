package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	mcp_server "github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/cache"
	"github.com/nimendra/ERPBridge/internal/connector"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/nimendra/ERPBridge/internal/mcp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var version = "dev"

func main() {
	stdioFlag := flag.Bool("stdio", false, "Run in STDIO transport mode")
	flag.Parse()

	transport := os.Getenv("MCP_TRANSPORT")
	useStdio := *stdioFlag || transport == "stdio"

	if useStdio {
		_ = os.Setenv("LOG_TO_STDERR", "true")
	}

	// Initialize Logger
	rootLog := logger.Init()

	slog.Info("Starting ERPBridge Server", slog.String("version", version), slog.Bool("stdio", useStdio))

	mcpPort := os.Getenv("MCP_PORT")
	if mcpPort == "" {
		mcpPort = "8080"
	}

	schemasDir := os.Getenv("SCHEMAS_DIR")
	if schemasDir == "" {
		schemasDir = "schemas"
	}

	redisURL := os.Getenv("REDIS_URL")

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
			slog.Error("failed to parse redis url", slog.String("error", err.Error()))
		} else {
			rdb := redis.NewClient(opt)
			cacheMgr = cache.NewManager(rdb, rootLog)
			if err := cacheMgr.EnsureIndex(context.Background()); err != nil {
				slog.Warn("failed to ensure redis index", slog.String("error", err.Error()))
			}
			slog.Info("cache initialized", slog.String("redis_url", redisURL))
		}
	}

	conn := connector.NewClient(rootLog)
	server := mcp.NewServer(conn, cacheMgr, rootLog)

	// Load tools from schemas directory
	loadTools(server, schemasDir)

	// Start Hot Reloading
	go watchSchemas(server, schemasDir)

	if useStdio {
		slog.Info("ERPBridge Server running in STDIO mode")
		if err := mcp_server.ServeStdio(server.MCPServer()); err != nil {
			slog.Error("stdio server failed", slog.String("error", err.Error()))
		}
		return
	}

	mux := http.NewServeMux()
	server.ServeHTTP(mux, baseURL)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	slog.Info("ERPBridge Server listening",
		slog.String("port", mcpPort),
		slog.String("mcp_http", baseURL+"/mcp/"),
	)
	log.Fatal(http.ListenAndServe(":"+mcpPort, mux))
}

func loadTools(s *mcp.Server, dir string) {
	slog.Info("loading tools", slog.String("directory", dir))
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			reloadTool(s, path)
		}
		return nil
	})
	if err != nil {
		log.Printf("error walking schemas directory: %v", err)
	}
}

func watchSchemas(s *mcp.Server, dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("failed to create watcher", slog.String("error", err.Error()))
		return
	}
	defer func() { _ = watcher.Close() }()

	// Add recursive directories
	if err := addRecursive(watcher, dir); err != nil {
		slog.Error("failed to start recursive watch", slog.String("error", err.Error()), slog.String("directory", dir))
		return
	}

	slog.Info("schema hot-reloading active", slog.String("directory", dir))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Handle directory creation for recursive watching
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					_ = addRecursive(watcher, event.Name)
					continue
				}
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if filepath.Ext(event.Name) == ".json" {
					slog.Info("schema change detected, reloading", slog.String("file", event.Name))
					reloadTool(s, event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", slog.String("error", err.Error()))
		}
	}
}

func addRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				return err
			}
			slog.Debug("watching directory", slog.String("path", path))
		}
		return nil
	})
}

func reloadTool(s *mcp.Server, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed to read schema", slog.String("path", path), slog.String("error", err.Error()))
		return
	}
	var tool mcp.Tool
	if err := json.Unmarshal(data, &tool); err != nil {
		slog.Error("failed to unmarshal schema", slog.String("path", path), slog.String("error", err.Error()))
		return
	}
	s.RegisterTool(&tool)
}
