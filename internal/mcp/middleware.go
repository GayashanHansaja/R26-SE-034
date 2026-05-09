package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/nimendra/ERPBridge/internal/metrics"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware provides per-session rate limiting for tool execution.
type RateLimitMiddleware struct {
	limiters map[string]*rate.Limiter
	mutex    sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimitMiddleware initializes a new RateLimitMiddleware with the given rate and burst.
func NewRateLimitMiddleware(requestsPerSecond float64, burst int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

func (m *RateLimitMiddleware) getLimiter(sessionID string) *rate.Limiter {
	if sessionID == "" {
		sessionID = "anonymous"
	}
	m.mutex.RLock()
	limiter, exists := m.limiters[sessionID]
	m.mutex.RUnlock()

	if !exists {
		m.mutex.Lock()
		limiter = rate.NewLimiter(m.rate, m.burst)
		m.limiters[sessionID] = limiter
		m.mutex.Unlock()
	}

	return limiter
}

// Handle returns a server.ToolHandlerMiddleware that enforces rate limits.
func (m *RateLimitMiddleware) Handle() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID := ""
			if sess := server.ClientSessionFromContext(ctx); sess != nil {
				sessionID = sess.SessionID()
			}
			limiter := m.getLimiter(sessionID)

			if !limiter.Allow() {
				return mcp.NewToolResultError(fmt.Sprintf("rate limit exceeded for session %s", sessionID)), nil
			}

			return next(ctx, req)
		}
	}
}

// LoggingMiddleware audits tool execution by logging start, completion, and failure events.
func LoggingMiddleware(log *slog.Logger) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()
			sessionID := ""
			if sess := server.ClientSessionFromContext(ctx); sess != nil {
				sessionID = sess.SessionID()
			}
			reqID := logger.NewRequestID()

			toolLog := log.With(
				slog.String("session_id", sessionID),
				slog.String("request_id", reqID),
				slog.String("tool_name", req.Params.Name),
			)

			ctx = logger.WithLogger(ctx, toolLog)

			toolLog.InfoContext(ctx, "tool execution started", slog.Any("arguments", req.Params.Arguments))

			result, err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				toolLog.ErrorContext(ctx, "tool execution failed",
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
				)
			} else {
				toolLog.InfoContext(ctx, "tool execution completed",
					slog.Duration("duration", duration),
				)
			}

			return result, err
		}
	}
}

// MetricsMiddleware records execution latency and invocation counts for Prometheus.
func MetricsMiddleware() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()
			result, err := next(ctx, req)
			duration := time.Since(start)

			status := "SUCCESS"
			if err != nil || (result != nil && result.IsError) {
				status = "ERROR"
			}

			metrics.ToolInvocationsTotal.WithLabelValues(req.Params.Name, status).Inc()
			metrics.ToolLatency.WithLabelValues(req.Params.Name).Observe(duration.Seconds())

			return result, err
		}
	}
}

// CacheMiddleware handles exact matching cache for tool results.
func (s *Server) CacheMiddleware(t *Tool) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if s.cache == nil || t.Cache == nil || !t.Cache.Enabled {
				return next(ctx, req)
			}

			args, _ := req.Params.Arguments.(map[string]any)
			role := "" // Extract from context if needed in future

			// READ from cache
			entry, err := s.cache.Get(ctx, t.Name, role, args, *t.Cache)
			if err == nil && entry != nil && entry.HitType != "miss" {
				metrics.CacheHitsTotal.WithLabelValues(entry.HitType).Inc()
				s.log.Debug("cache hit", slog.String("tool", t.Name), slog.String("type", entry.HitType))
				return mcp.NewToolResultText(string(entry.Response)), nil
			}

			metrics.CacheMissesTotal.Inc()

			// Execute next
			result, err := next(ctx, req)
			if err != nil || result == nil || result.IsError {
				return result, err
			}

			// WRITE to cache
			respJSON, _ := json.Marshal(result.Content)
			if err := s.cache.Set(ctx, t.Name, role, args, respJSON, *t.Cache); err != nil {
				s.log.Warn("failed to cache result", slog.String("error", err.Error()))
			}

			// Invalidation (Auto-Flush)
			if len(t.Cache.FlushOn) > 0 {
				if err := s.cache.AutoFlush(ctx, t.Cache.FlushOn); err != nil {
					s.log.Warn("auto-flush failed", slog.String("error", err.Error()))
				}
			}

			return result, err
		}
	}
}
