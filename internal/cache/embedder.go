// internal/cache/embedder.go
package cache

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type HFEmbedder struct {
    baseURL string
    client  *http.Client
}

func NewHFEmbedder(baseURL string) *HFEmbedder {
    return &HFEmbedder{baseURL: baseURL, client: &http.Client{}}
}

func (e *HFEmbedder) Dim() int { return 384 }

func (e *HFEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    body, _ := json.Marshal(map[string]string{"inputs": text})
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        e.baseURL+"/embed", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := e.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("embed request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("embed request failed with status: %s", resp.Status)
    }

    var result [][]float32
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("embed decode: %w", err)
    }
    if len(result) == 0 {
        return nil, fmt.Errorf("empty embedding response")
    }
    return result[0], nil
}
