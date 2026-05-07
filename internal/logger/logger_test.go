package logger

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"WARN", slog.LevelWarn},
		{"Error", slog.LevelError},
		{"info", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		if got := parseLevel(tt.input); got != tt.expected {
			t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNewRequestID(t *testing.T) {
	id1 := NewRequestID()
	id2 := NewRequestID()

	if id1 == id2 {
		t.Errorf("Expected unique IDs, got %v and %v", id1, id2)
	}

	if len(id1) < 5 {
		t.Errorf("Expected ID to have prefix and hex suffix, got %v", id1)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	ch := Subscribe()
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}

	Unsubscribe(ch)

	// Verify channel is closed
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed after Unsubscribe")
	}
}

func TestBroadcastHandler(t *testing.T) {
	// Reset global state for test
	listenersMu.Lock()
	logListeners = nil
	logBuffer = nil
	listenersMu.Unlock()

	ch := Subscribe()
	defer Unsubscribe(ch)

	l := Init()
	l.Info("test message")

	// Check buffer
	recent := GetRecentLogs()
	if len(recent) != 1 {
		t.Errorf("Expected 1 log in buffer, got %d", len(recent))
	}

	// Check broadcast
	select {
	case msg := <-ch:
		if len(msg) == 0 {
			t.Error("Received empty message from subscriber")
		}
	default:
		t.Error("Subscriber did not receive the broadcast message")
	}
}

func TestComponent(t *testing.T) {
	root := slog.Default()
	comp := Component(root, "test-comp")

	if comp == nil {
		t.Fatal("Component returned nil logger")
	}

	// Test override via environment
	t.Setenv("LOG_LEVEL_OVERRIDE", "debug")

	compOverride := Component(root, "override")
	if compOverride == nil {
		t.Fatal("Component with override returned nil logger")
	}
}
