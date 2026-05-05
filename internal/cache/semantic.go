// internal/cache/semantic.go
package cache

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "math"
    "strings"
    "time"

    "github.com/google/uuid"
)

// Embedder is the interface the cache uses to produce vector embeddings.
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    Dim() int
}

func (m *Manager) semanticGet(ctx context.Context, tool, roleKey string, argsJSON []byte, threshold float32) (*Entry, float32, error) {
    embedding, err := m.embedder.Embed(ctx, string(argsJSON))
    if err != nil {
        return nil, 0, err
    }

    query := fmt.Sprintf(
        "(@tool:{%s} @role:{%s})=>[KNN 1 @args_emb $vec AS score]",
        escapeTag(tool), escapeTag(roleKey),
    )

    res, err := m.rdb.Do(ctx, "FT.SEARCH", "idx:semantic", query,
        "PARAMS", "2", "vec", float32ToBytes(embedding),
        "SORTBY", "score",
        "LIMIT", "0", "1",
        "RETURN", "3", "response", "created", "score",
        "DIALECT", "2",
    ).Result()
    if err != nil {
        return nil, 0, err
    }

    // Parse FT.SEARCH response
    // Format: [count, key1, [field1, val1, field2, val2, ...], key2, ...]
    data, ok := res.([]any)
    if !ok || len(data) < 3 {
        return nil, 0, nil
    }

    count, ok := data[0].(int64)
    if !ok || count == 0 {
        return nil, 0, nil
    }

    fields, ok := data[2].([]any)
    if !ok {
        return nil, 0, nil
    }

    var score float32
    var response string
    var created int64

    for i := 0; i < len(fields); i += 2 {
        key, _ := fields[i].(string)
        val := fields[i+1]
        switch key {
        case "score":
            s, _ := val.(string)
            fmt.Sscanf(s, "%f", &score)
        case "response":
            response, _ = val.(string)
        case "created":
            c, _ := val.(string)
            fmt.Sscanf(c, "%d", &created)
        }
    }

    // Cosine distance → similarity: similarity = 1 - distance
    similarity := float32(1) - score
    if similarity < threshold {
        return nil, similarity, nil // below threshold
    }

    return &Entry{
        Response: json.RawMessage(response),
        CachedAt: time.Unix(created, 0),
    }, similarity, nil
}

func (m *Manager) semanticSet(ctx context.Context, tool, roleKey string, argsJSON, response json.RawMessage, embedding []float32, ttl time.Duration) error {
    id := fmt.Sprintf("sem:%s", uuid.New().String())
    return m.rdb.HSet(ctx, id,
        "tool",     tool,
        "role",     roleKey,
        "args_raw", string(argsJSON),
        "response", string(response),
        "args_emb", float32ToBytes(embedding),
        "created",  time.Now().Unix(),
        "ttl",      int(ttl.Seconds()),
    ).Err()
}

func float32ToBytes(v []float32) []byte {
    buf := make([]byte, len(v)*4)
    for i, f := range v {
        binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
    }
    return buf
}

func escapeTag(s string) string {
    return strings.ReplaceAll(s, "-", "\\-")
}
