# AI Agent Integration Guide

This middleware exposes ERP functionality via the **Model Context Protocol (MCP)** using multiple transport layers.

## MCP Endpoints

### 1. Streamable HTTP (Recommended for Postman/Modern)
- **Base URL**: `http://localhost:8080/mcp/`
- **Transport**: MCP 2025-03-26

### 2. Stdio (Recommended for Claude/Cursor)
- **Method**: Start `bridgectl` with the `--stdio` flag.
- **Transport**: Standard Input/Output.

## Connectivity Guide
For detailed configuration, session management, and Postman setup, see the [Connectivity & Transport Guide](./docs/connectivity.md).

## Integration Patterns

1. **Transport Selection**: Local agents should use **Stdio** for best performance. Remote or web-based agents should use **Streamable HTTP**.
2. **Tool Discovery**: Once the connection is established, the agent can request the list of available tools via the standard MCP `initialize` and `tools/list` lifecycle.
3. **Execution**: The middleware routes tool calls to the underlying ERP systems, handling resilience and caching automatically.

## Example Discovery

Agents using the standard MCP Stdio client should be configured to run:
`bridgectl serve --stdio`

Modern HTTP clients (like Postman) should use:
`http://localhost:8080/mcp/`

## CLI Access
Agents can also use the `bridgectl` binary for local or containerized execution:
```bash
./bridgectl tool invoke finance.invoices '{"page": 1}' -o json
```
