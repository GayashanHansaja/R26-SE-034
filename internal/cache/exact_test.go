package cache

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/nimendra/ERPBridge/internal/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_ExactGet(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	log := logger.Init()
	m := NewManager(rdb, log)

	ctx := context.Background()

	t.Run("miss", func(t *testing.T) {
		entry, err := m.exactGet(ctx, "non-existent")
		assert.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("hit", func(t *testing.T) {
		err := s.Set("exact:test-tool:admin:123", `{"status":"ok"}`)
		require.NoError(t, err)

		entry, err := m.exactGet(ctx, "exact:test-tool:admin:123")
		assert.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, json.RawMessage(`{"status":"ok"}`), entry.Response)
		assert.NotZero(t, entry.CachedAt)
	})

	t.Run("redis error", func(t *testing.T) {
		// Close redis to force an error
		s.Close()

		_, err := m.exactGet(ctx, "some-key")
		assert.Error(t, err)
	})
}
