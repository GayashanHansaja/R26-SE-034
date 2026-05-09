// internal/cache/manager_test.go
package cache

import (
	"testing"

	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestArgsHash(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "empty args",
			args: map[string]any{},
			want: "empty",
		},
		{
			name: "simple args",
			args: map[string]any{"a": 1, "b": "2"},
			want: "347158d2", // Pre-calculated for these args
		},
		{
			name: "reordered args",
			args: map[string]any{"b": "2", "a": 1},
			want: "347158d2", // Should be the same
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := argsHash(tt.args); got != tt.want {
				t.Errorf("argsHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoleScope(t *testing.T) {
	tests := []struct {
		role       string
		isReadOnly bool
		want       string
	}{
		{"admin", false, "admin"},
		{"", false, "anonymous"},
		{"user", true, "shared"},
	}

	for _, tt := range tests {
		got := roleScope(tt.role, tt.isReadOnly)
		assert.Equal(t, tt.want, got)
	}
}

func TestExactKey(t *testing.T) {
	key := exactKey("tool", "role", map[string]any{"a": 1})
	assert.Contains(t, key, "exact:tool:role:")
}

func TestNewManager(t *testing.T) {
	log := logger.Init()
	m := NewManager(nil, log)
	assert.NotNil(t, m)
}
