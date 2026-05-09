# Connectivity & Transport Guide

ERPBridge supports multiple transport protocols to ensure compatibility with a wide range of clients, from modern AI agents and IDEs to standard developer tools like Postman.

## 1. Streamable HTTP (Modern MCP)
This is the latest transport specification (March 2025) designed for stateless or web-friendly environments. It is the recommended way to connect **Postman** and other modern MCP clients.

- **Base URL:** `http://localhost:8080/mcp/`
- **Handshake:** `POST /mcp/initialize`
- **Transport Specification:** MCP 2025-03-26
- **Features:**
    - Request/Response via standard POST.
    - Session management via `Mcp-Session-Id` header.
    - Full CORS support for browser and desktop clients.

### Postman Configuration
- **Transport Type:** Streamable HTTP
- **URL:** `http://localhost:8080/mcp/`

## 2. Stdio (Local Integration)
Standard Input/Output (stdio) is the preferred transport for local integrations where the ERPBridge server is started as a child process by the client (e.g., an IDE or a CLI tool).

- **Best For:** Claude Desktop, Cursor, and other IDE-integrated agents running locally.
- **Usage:** Run the server with the `--stdio` flag.

## 3. Direct API (Internal/CLI)
The server exposes direct HTTP endpoints for internal management, performance monitoring, and the `bridgectl` CLI. These do not require a full MCP handshake.

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/tools/invoke` | `POST` | Directly invoke an MCP tool. |
| `/api/cache/stats` | `GET` | Retrieve tool cache performance metrics. |
| `/api/cache/flush` | `POST` | Flush specific or all cache entries. |
| `/api/logs/stream` | `GET` | Real-time structured log stream (SSE). |
| `/api/logs/recent` | `GET` | Fetch recent log history in JSON format. |

## 4. Protection & Limits
ERPBridge includes built-in protection for underlying ERP systems.

- **Rate Limiting:** Request throttling is enforced per-session (token bucket).
- **Default Limits:** 5 requests per second with a burst of 10 (configurable via environment variables).

## 5. Monitoring & Health
Standard endpoints for system health and observability.

- **Health Check:** `GET /mcp/health` (Returns `{"status": "ok"}`)
- **Metrics:** `GET /metrics` (Prometheus formatted metrics)

## Summary Table

| Client Type | Recommended Transport | Base URL / Method |
| :--- | :--- | :--- |
| **Postman / Web** | Streamable HTTP | `http://localhost:8080/mcp/` |
| **Claude / Cursor** | Stdio | `bridgectl serve --stdio` |
| **bridgectl / Scripts** | Direct API | `http://localhost:8080/api/` |
| **Prometheus** | HTTP | `http://localhost:8080/metrics` |
