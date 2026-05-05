// internal/cache/flush.go
package cache

import (
    "context"
    "fmt"
    "strings"
)

// FlushTool removes all exact and semantic cache entries for a given tool name.
func (m *Manager) FlushTool(ctx context.Context, toolName string) (int, error) {
    total := 0

    // 1. Flush exact cache entries (pattern scan)
    pattern := fmt.Sprintf("exact:%s:*", toolName)
    deleted, err := m.scanAndDelete(ctx, pattern)
    if err != nil {
        return total, fmt.Errorf("flush exact: %w", err)
    }
    total += deleted

    // 2. Flush semantic entries (tag filter)
    query := fmt.Sprintf("@tool:{%s}", escapeTag(toolName))
    keys, err := m.searchKeys(ctx, query)
    if err != nil {
        return total, fmt.Errorf("flush semantic search: %w", err)
    }
    if len(keys) > 0 {
        deleted, err := m.rdb.Del(ctx, keys...).Result()
        if err != nil {
            return total, fmt.Errorf("flush semantic delete: %w", err)
        }
        total += int(deleted)
    }

    return total, nil
}

// FlushModule removes all cache entries for every tool in a module.
func (m *Manager) FlushModule(ctx context.Context, module string) (int, error) {
    pattern := fmt.Sprintf("exact:%s.*:*", module)
    exact, err := m.scanAndDelete(ctx, pattern)
    if err != nil {
        return 0, err
    }
    query := fmt.Sprintf("@tool:{%s\\..*}", module) // Redis tag regex
    keys, _ := m.searchKeys(ctx, query)
    semantic := 0
    if len(keys) > 0 {
        n, _ := m.rdb.Del(ctx, keys...).Result()
        semantic = int(n)
    }
    return exact + semantic, nil
}

// AutoFlush is called by the MCP server after a successful write tool response.
func (m *Manager) AutoFlush(ctx context.Context, flushOn []string) error {
    for _, toolName := range flushOn {
        if _, err := m.FlushTool(ctx, toolName); err != nil {
            fmt.Printf("warn: auto-flush failed for %s: %v\n", toolName, err)
        }
    }
    return nil
}

func (m *Manager) scanAndDelete(ctx context.Context, pattern string) (int, error) {
    var count int
    iter := m.rdb.Scan(ctx, 0, pattern, 100).Iterator()
    for iter.Next(ctx) {
        m.rdb.Del(ctx, iter.Val())
        count++
    }
    return count, iter.Err()
}

func (m *Manager) searchKeys(ctx context.Context, query string) ([]string, error) {
    res, err := m.rdb.Do(ctx, "FT.SEARCH", "idx:semantic", query, "NOCONTENT").Result()
    if err != nil {
        // Index might not exist yet if no semantic entries were added
        if strings.Contains(err.Error(), "no such index") || strings.Contains(err.Error(), "Unknown index name") {
            return nil, nil
        }
        return nil, err
    }

    data, ok := res.([]any)
    if !ok || len(data) < 2 {
        return nil, nil
    }

    count, _ := data[0].(int64)
    if count == 0 {
        return nil, nil
    }

    keys := make([]string, 0, count)
    for i := 1; i < len(data); i++ {
        key, _ := data[i].(string)
        keys = append(keys, key)
    }

    return keys, nil
}
