// internal/cache/manager.go
package cache

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "log/slog"
    "sort"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/nimendra/ERPBridge/internal/logger"
)

type Config struct {
    Enabled           bool     `json:"enabled"`
    TTLSeconds        int      `json:"ttlSeconds"`
    SemanticThreshold float32  `json:"semanticThreshold"`
    IsReadOnly        bool     `json:"isReadOnly"` // true = shared cache; false = role-isolated
    FlushOn           []string `json:"flushOn"`
}

type Entry struct {
    Response  json.RawMessage
    CachedAt  time.Time
    HitType   string // "exact" | "semantic" | "miss"
}

type Manager struct {
    rdb      *redis.Client
    embedder Embedder // interface — swappable model
    log      *slog.Logger
}

func NewManager(rdb *redis.Client, embedder Embedder, rootLog *slog.Logger) *Manager {
    return &Manager{
        rdb:      rdb,
        embedder: embedder,
        log:      logger.Component(rootLog, "cache"),
    }
}

// EnsureIndex creates the RediSearch vector index if it doesn't exist.
func (m *Manager) EnsureIndex(ctx context.Context) error {
    // Check if index exists
    _, err := m.rdb.Do(ctx, "FT.INFO", "idx:semantic").Result()
    if err == nil {
        return nil // Index already exists
    }

    // Create index
    // FT.CREATE idx:semantic ON HASH PREFIX 1 "sem:" SCHEMA tool TAG role TAG args_emb VECTOR HNSW 6 TYPE FLOAT32 DIM 768 DISTANCE_METRIC COSINE
    err = m.rdb.Do(ctx, "FT.CREATE", "idx:semantic",
        "ON", "HASH",
        "PREFIX", "1", "sem:",
        "SCHEMA",
        "tool", "TAG",
        "role", "TAG",
        "args_emb", "VECTOR", "HNSW", "6",
        "TYPE", "FLOAT32",
        "DIM", fmt.Sprintf("%d", m.embedder.Dim()),
        "DISTANCE_METRIC", "COSINE",
    ).Err()

    return err
}

// Get tries exact match first, then semantic fallback.
// Returns nil entry on a full miss.
func (m *Manager) Get(ctx context.Context, tool, role string, args map[string]any, cfg Config) (*Entry, error) {
    if !cfg.Enabled {
        return &Entry{HitType: "miss"}, nil
    }

    log := logger.FromContext(ctx)
    roleKey := roleScope(role, cfg.IsReadOnly)

    // Layer 1 — exact match
    key := exactKey(tool, roleKey, args)
    if entry, err := m.exactGet(ctx, key); err == nil && entry != nil {
        entry.HitType = "exact"
        log.Info("cache hit", slog.String("type", "exact"), slog.String("key", key))
        return entry, nil
    }

    // Layer 2 — semantic fallback
    if cfg.SemanticThreshold > 0 && m.embedder != nil {
        argsJSON, _ := json.Marshal(args)
        if entry, score, err := m.semanticGet(ctx, tool, roleKey, argsJSON, cfg.SemanticThreshold); err == nil && entry != nil {
            entry.HitType = "semantic"
            log.Info("cache hit", slog.String("type", "semantic"), slog.Float64("score", float64(score)))
            return entry, nil
        }
    }

    log.Info("cache miss", slog.String("exact_key", key))
    return &Entry{HitType: "miss"}, nil
}

// Set stores a response in both the exact and semantic caches.
func (m *Manager) Set(ctx context.Context, tool, role string, args map[string]any, response json.RawMessage, cfg Config) error {
    if !cfg.Enabled {
        return nil
    }

    log := logger.FromContext(ctx)
    roleKey := roleScope(role, cfg.IsReadOnly)
    ttl := time.Duration(cfg.TTLSeconds) * time.Second
    argsJSON, _ := json.Marshal(args)

    // Exact cache
    key := exactKey(tool, roleKey, args)
    if err := m.rdb.Set(ctx, key, response, ttl).Err(); err != nil {
        log.Error("redis error", slog.String("operation", "SET"), slog.String("error", err.Error()))
        return fmt.Errorf("exact cache set: %w", err)
    }
    log.Debug("cache stored", slog.String("key", key), slog.Int("ttl_seconds", cfg.TTLSeconds))

    // Semantic cache
    if cfg.SemanticThreshold > 0 && m.embedder != nil {
        embedding, err := m.embedder.Embed(ctx, string(argsJSON))
        if err != nil {
            // Non-fatal — exact cache still works
            return nil
        }
        return m.semanticSet(ctx, tool, roleKey, argsJSON, response, embedding, ttl)
    }
    
    return nil
}

// --- helpers ---

func exactKey(tool, roleKey string, args map[string]any) string {
    return fmt.Sprintf("exact:%s:%s:%s", tool, roleKey, argsHash(args))
}

func argsHash(args map[string]any) string {
    if len(args) == 0 {
        return "empty"
    }
    // Sort keys for canonical JSON before hashing
    keys := make([]string, 0, len(args))
    for k := range args {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    ordered := make([]mapEntry, 0, len(args))
    for _, k := range keys {
        ordered = append(ordered, mapEntry{Key: k, Value: args[k]})
    }
    b, _ := json.Marshal(ordered)
    h := sha256.Sum256(b)
    return fmt.Sprintf("%x", h[:4]) // 8 hex chars
}

type mapEntry struct {
    Key   string `json:"k"`
    Value any    `json:"v"`
}

func roleScope(role string, isReadOnly bool) string {
    if isReadOnly {
        return "shared"
    }
    if role == "" {
        return "anonymous"
    }
    return role
}
