# Semantic & Exact Caching

ERPBridge implements a high-performance, layered caching strategy designed to minimize latency and reduce load on legacy ERP systems. By combining traditional exact-match caching with AI-driven semantic search, ERPBridge can serve complex ERP data in milliseconds.

## 🚀 Overview

The caching mechanism operates as a middleware layer in the MCP tool execution pipeline. It uses **Redis Stack** as the backend for both key-value storage and vector search.

### 1. Layer 1: Exact Match (SHA256)
The first layer provides O(1) lookups for identical requests.
- **Key Generation**: A deterministic hash of `tool_name` + `user_role` + `json_sorted_arguments`.
- **TTL**: Configurable per tool (default: 3600s).
- **Benefit**: Extremely fast response times (<1ms) for repetitive queries.

### 2. Layer 2: Semantic Fallback (Vector Search)
If no exact match is found, ERPBridge can search for semantically similar previous queries.
- **Mechanism**: JSON arguments are transformed into a 384-dimensional vector using a local embedding service (`sentence-transformers/all-MiniLM-L6-v2`).
- **Vector Index**: Uses Redis HNSW (Hierarchical Navigable Small World) index for fast approximate nearest neighbor search.
- **Threshold**: Only results with a similarity score (1 - cosine distance) above the `semanticThreshold` are served.
- **Benefit**: Understands intent. For example, `"get items for finance"` will hit the cache for `"list all finance department items"`.

---

## ⚙️ Configuration

Caching is opt-in and configured per-tool in the JSON schema files located in `schemas/`.

### Example Schema Configuration

```json
{
  "name": "erp.GET-resource-Item",
  "cache": {
    "enabled": true,
    "ttlSeconds": 3600,
    "semanticThreshold": 0.85,
    "isReadOnly": true,
    "flushOn": []
  }
}
```

| Field | Description |
| :--- | :--- |
| `enabled` | Enables/disables the cache middleware for this tool. |
| `ttlSeconds` | How long the entry remains in Redis. |
| `semanticThreshold` | Similarity score (0.0 - 1.0) required for a semantic hit. Set to `0` to disable semantic fallback. |
| `isReadOnly` | If `true`, the cache is shared globally (`role: shared`). If `false`, entries are isolated by the user's MCP role. |
| `flushOn` | An array of tool names. When the current tool is executed, it automatically flushes the cache for the listed tools. |

---

## 🧹 Cache Invalidation (Auto-Flush)

To prevent stale data, ERPBridge supports automatic cache invalidation. This is typically used on `POST`, `PUT`, or `PATCH` tools.

**Example: Invalidating "Get Invoices" when a new one is created.**
In `erp.POST-resource-Purchase Invoice.json`:
```json
"cache": {
  "enabled": false,
  "flushOn": ["erp.GET-resource-Purchase Invoice"]
}
```

---

## 🛠 Management with `bridgectl`

The developer CLI provides tools to monitor and manage the cache.

### Check Cache Statistics
Provides counts of exact/semantic keys and memory usage.
```bash
bridgectl cache stats
```

### Flush Cache
Manually clear the cache for a specific tool or an entire module.
```bash
# Flush specific tool
bridgectl cache flush --tool erp.GET-resource-Item

# Flush entire module
bridgectl cache flush --module erp

# Flush everything
bridgectl cache flush --all
```

---

## 🏗 System Architecture

1. **ERPBridge Server**: Orchestrates the middleware and talks to Redis.
2. **Redis Stack**: Stores the hashes and performs vector similarity search.
3. **Embedder Service**: A containerized HuggingFace Inference API (`text-embeddings-inference`) that generates vectors.

For deployment details, see the [Docker Guide](./docker.md).
