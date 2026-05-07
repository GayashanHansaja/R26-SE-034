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

## 2. SSE Transport (Classic MCP)
Server-Sent Events (SSE) is the "classic" persistent connection method used by most current AI tools and IDE plugins.

- **SSE Endpoint:** `http://localhost:8080/mcp/sse`
- **Message Endpoint:** `http://localhost:8080/mcp/messages`
- **Transport Specification:** MCP 2024-11-05
- **Best For:** Claude Desktop, Cursor, and other IDE-integrated agents.

### Integration Flow
1. Client connects via `GET /mcp/sse` to establish a stream.
2. Server responds with a session ID and the message endpoint URL.
3. Client sends JSON-RPC requests via `POST` to the message endpoint.

## 3. Direct API (Internal/CLI)
The server exposes direct HTTP endpoints for internal management, performance monitoring, and the `bridgectl` CLI. These do not require a full MCP handshake.

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/tools/invoke` | `POST` | Directly invoke an MCP tool. |
| `/api/cache/stats` | `GET` | Retrieve semantic cache performance metrics. |
| `/api/cache/flush` | `POST` | Flush specific or all cache entries. |
| `/api/logs/stream` | `GET` | Real-time structured log stream (SSE). |
| `/api/logs/recent` | `GET` | Fetch recent log history in JSON format. |

## 4. Monitoring & Health
Standard endpoints for system health and observability.

- **Health Check:** `GET /mcp/health` (Returns `{"status": "ok"}`)
- **Metrics:** `GET /metrics` (Prometheus formatted metrics)

## Summary Table

| Client Type | Recommended Transport | Base URL |
| :--- | :--- | :--- |
| **Postman / Web** | Streamable HTTP | `http://localhost:8080/mcp/` |
| **Claude / Cursor** | SSE | `http://localhost:8080/mcp/sse` |
| **bridgectl / Scripts** | Direct API | `http://localhost:8080/api/` |
| **Prometheus** | HTTP | `http://localhost:8080/metrics` |
