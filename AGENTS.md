# AI Agent Integration Guide

This middleware exposes ERP functionality via the **Model Context Protocol (MCP)** using multiple transport layers.

## MCP Endpoints

### 1. Streamable HTTP (Recommended for Postman/Modern)
- **Base URL**: `http://localhost:8080/mcp/`
- **Transport**: MCP 2025-03-26

### 2. SSE (Classic for Claude/Cursor)
- **SSE Connection**: `GET /mcp/sse`
- **Post Messages**: `POST /mcp/messages`

## Connectivity Guide
For detailed configuration, session management, and Postman setup, see the [Connectivity & Transport Guide](./docs/connectivity.md).

## Integration Patterns

1. **SSE Handshake**: Agents should connect to `/mcp/sse` to establish a persistent connection. The server will provide a session ID and a dedicated message endpoint.
2. **Tool Discovery**: Once connected, the agent can request the list of available tools.
3. **Execution**: Use the provided message endpoint to call tools. The middleware routes these calls to the underlying ERP systems.

## Example Discovery (mark3labs/mcp-go compatible)

Agents using the standard MCP SSE client should point to:
`http://localhost:8080/mcp/sse`

Modern clients (like Postman) should use:
`http://localhost:8080/mcp/`

## CLI Access
Agents can also use the `bridgectl` binary for local or containerized execution:
```bash
./bridgectl tool invoke finance.invoices '{"page": 1}' -o json
```
