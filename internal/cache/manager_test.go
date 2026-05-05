// internal/cache/manager_test.go
package cache

import (
    "testing"
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
