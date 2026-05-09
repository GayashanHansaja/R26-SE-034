package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nimendra/ERPBridge/internal/metrics"
)

// TelemetryHooks implements lifecycle callbacks for telemetry and logging.
type TelemetryHooks struct {
	logger         *slog.Logger
	callStartTimes sync.Map
	activeSessions atomic.Int32
}

// NewTelemetryHooks creates a new TelemetryHooks instance.
func NewTelemetryHooks(logger *slog.Logger) *TelemetryHooks {
	return &TelemetryHooks{
		logger: logger,
	}
}

// Register attaches the telemetry hooks to the given MCP server.
func (h *TelemetryHooks) Register(s *server.MCPServer) {
	hooks := s.GetHooks()
	if hooks == nil {
		return
	}

	hooks.AddOnRegisterSession(h.OnSessionStart)
	hooks.AddOnUnregisterSession(h.OnSessionEnd)
	hooks.AddBeforeCallTool(h.OnBeforeCallTool)
	hooks.AddAfterCallTool(h.OnAfterCallTool)
}

func (h *TelemetryHooks) OnServerStart() {
	h.logger.Info("MCP Server starting")
	metrics.ServerStartsTotal.Inc()
}

func (h *TelemetryHooks) OnServerStop() {
	h.logger.Info("MCP Server stopping")
	metrics.ServerStopsTotal.Inc()
}

func (h *TelemetryHooks) OnSessionStart(ctx context.Context, session server.ClientSession) {
	h.logger.Info("Session started", slog.String("session_id", session.SessionID()))
	metrics.SessionsStartedTotal.Inc()
	h.activeSessions.Add(1)
	metrics.SessionsActive.Set(float64(h.activeSessions.Load()))
}

func (h *TelemetryHooks) OnSessionEnd(ctx context.Context, session server.ClientSession) {
	h.logger.Info("Session ended", slog.String("session_id", session.SessionID()))
	metrics.SessionsEndedTotal.Inc()
	h.activeSessions.Add(-1)
	metrics.SessionsActive.Set(float64(h.activeSessions.Load()))
}

func (h *TelemetryHooks) OnBeforeCallTool(ctx context.Context, id any, message *mcp.CallToolRequest) {
	h.callStartTimes.Store(id, time.Now())
}

func (h *TelemetryHooks) OnAfterCallTool(ctx context.Context, id any, message *mcp.CallToolRequest, result any) {
	start, ok := h.callStartTimes.LoadAndDelete(id)
	if !ok {
		return
	}
	duration := time.Since(start.(time.Time))

	h.logger.Info("Tool call completed",
		slog.String("tool", message.Params.Name),
		slog.Duration("duration", duration),
	)
}

// BusinessHooks implements custom business logic callbacks.
type BusinessHooks struct {
	notifier *CustomNotifier
	logger   *slog.Logger
}

// NewBusinessHooks creates a new BusinessHooks instance.
func NewBusinessHooks(notifier *CustomNotifier, logger *slog.Logger) *BusinessHooks {
	return &BusinessHooks{
		notifier: notifier,
		logger:   logger,
	}
}

// Register attaches the business logic hooks to the given MCP server.
func (h *BusinessHooks) Register(s *server.MCPServer) {
	hooks := s.GetHooks()
	if hooks == nil {
		return
	}

	hooks.AddOnError(h.OnError)
	hooks.AddOnRegisterSession(h.OnSessionStart)
}

func (h *BusinessHooks) OnSessionStart(ctx context.Context, session server.ClientSession) {
	h.logger.Info("Business logic initialized for session", slog.String("session_id", session.SessionID()))
	// Send a welcome progress message
	h.notifier.SendProgress(ctx, 100, 100, "Connected to ERPBridge V2. Ready for declarative tool management.")
}

func (h *BusinessHooks) OnError(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
	h.logger.Error("Operation failed", slog.Any("method", method), slog.Any("error", err))

	// Send an alert on tool failure
	if string(method) == "tools/call" {
		h.notifier.SendAlert(ctx, fmt.Sprintf("Tool execution failed: %v", err), "error")
	}
}
