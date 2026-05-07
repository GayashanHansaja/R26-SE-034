package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/stretchr/testify/assert"
)

type mockSession struct {
	notifications chan mcp.JSONRPCNotification
}

func (m *mockSession) Initialize()       {}
func (m *mockSession) Initialized() bool { return true }
func (m *mockSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return m.notifications
}
func (m *mockSession) SessionID() string { return "mock-session" }

func TestServer_NotificationsAndLogs(t *testing.T) {
	// 1. Setup Logger
	rootLog := logger.Init()

	// 2. Setup Server
	s := NewServer(nil, nil, rootLog)

	// 3. Setup Mock Session
	mSess := &mockSession{
		notifications: make(chan mcp.JSONRPCNotification, 10),
	}
	ctx := s.mcpServer.WithContext(context.Background(), mSess)

	// 4. Invoke system.progress_test tool
	req := mcp.CallToolRequest{}
	req.Params.Name = "system.progress_test"
	req.Params.Arguments = map[string]any{"steps": float64(2)}

	// We need to find the handler that was registered
	mcpTool := s.mcpServer.ListTools()["system.progress_test"]
	assert.NotNil(t, mcpTool)

	// Call the handler directly
	_, err := mcpTool.Handler(ctx, req)
	assert.NoError(t, err)

	// 5. Verify Notifications
	select {
	case n := <-mSess.notifications:
		assert.Equal(t, "notifications/progress", n.Method)
		p := n.Params.AdditionalFields["progress"]
		assert.Equal(t, 1, p)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for notification 1")
	}

	select {
	case n := <-mSess.notifications:
		assert.Equal(t, "notifications/progress", n.Method)
		p := n.Params.AdditionalFields["progress"]
		assert.Equal(t, 2, p)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for notification 2")
	}

	// 6. Verify Logs using RecentLogs
	time.Sleep(100 * time.Millisecond) // Give a moment for logs to buffer
	logs := logger.GetRecentLogs()

	foundStart := false
	foundComplete := false

	for _, msg := range logs {
		var entry map[string]any
		if err := json.Unmarshal(msg, &entry); err != nil {
			continue
		}
		if entry["tool_name"] == "system.progress_test" {
			if entry["msg"] == "tool execution started" {
				foundStart = true
			}
			if entry["msg"] == "tool execution completed" {
				foundComplete = true
			}
		}
	}

	assert.True(t, foundStart, "start log not found in recent logs")
	assert.True(t, foundComplete, "complete log not found in recent logs")
}
