# Exact Match Caching

ERPBridge implements a high-performance caching layer designed to minimize latency and reduce load on legacy ERP systems. By serving repetitive queries directly from Redis, ERPBridge can deliver complex ERP data in milliseconds.

## 🚀 Overview

The caching mechanism operates as a middleware layer in the MCP tool execution pipeline. It uses **Redis** as the high-speed backend for key-value storage.

### Layer 1: Exact Match (SHA256)
The system provides O(1) lookups for identical requests.
- **Key Generation**: A deterministic hash of `tool_name` + `user_role` + `json_sorted_arguments`.
- **TTL**: Configurable per tool (default: 3600s).
- **Benefit**: Extremely fast response times (<1ms) for repetitive queries.

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
    "isReadOnly": true,
    "flushOn": []
  }
}
```

| Field | Description |
| :--- | :--- |
| `enabled` | Enables/disables the cache middleware for this tool. |
| `ttlSeconds` | How long the entry remains in Redis. |
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
Provides counts of cached keys and memory usage.
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
2. **Redis**: Stores the hashes and responses for high-speed retrieval.

For deployment details, see the [Docker Guide](./docker.md).
