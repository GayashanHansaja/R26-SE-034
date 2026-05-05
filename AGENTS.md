# AI Agent Integration Guide

This middleware exposes ERP functionality via the **Model Context Protocol (MCP)** using the **SSE (Server-Sent Events)** transport.

## MCP Endpoints

- **SSE Connection**: `GET /mcp/sse`
- **Post Messages**: `POST /mcp/messages` (used by the protocol to send requests after SSE handshake)

## Integration Patterns

1. **SSE Handshake**: Agents should connect to `/mcp/sse` to establish a persistent connection. The server will provide a session ID and a dedicated message endpoint.
2. **Tool Discovery**: Once connected, the agent can request the list of available tools.
3. **Execution**: Use the provided message endpoint to call tools. The middleware routes these calls to the underlying ERP systems.

## Example Discovery (mark3labs/mcp-go compatible)

Agents using the standard MCP SSE client should point to:
`http://localhost:8080/mcp/sse`

## CLI Access
Agents can also use the `bridgectl` binary for local or containerized execution:
```bash
./bridgectl tool invoke finance.invoices '{"page": 1}' -o json
```
