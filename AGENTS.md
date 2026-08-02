# AI Agent Integration Guide

This middleware exposes ERP functionality via the **Model Context Protocol (MCP)**.
It supports multiple transport layers.

## MCP Endpoints

### 1. Streamable HTTP (for Postman and remote clients)

- **Base URL**: `http://localhost:8080/mcp/`
- **Transport**: MCP 2025-03-26 (negotiation supports newer versions)

### 2. Stdio (for Claude and Cursor)

- **Method**: Run the server binary with the `--stdio` flag.
- **Transport**: Standard Input/Output.

## Connectivity Guide

For detailed configuration, session management, and Postman setup, see the [Connectivity & Transport Guide](./docs/connectivity.md).

## Integration Patterns

1. **Transport Selection**: Use **Stdio** for local agents. Use **Streamable HTTP** for remote or web-based agents.
2. **Tool Discovery**: After the connection is established, request the tool list via the standard MCP `initialize` and `tools/list` lifecycle.
3. **Execution**: The middleware routes tool calls to the underlying ERP systems. It handles resilience and caching automatically.

## Example Discovery

Run the server binary in Stdio mode when you use an MCP Stdio client:

```bash
erpbridge-server --stdio
```

Modern HTTP clients (like Postman) use:

```bash
http://localhost:8080/mcp/
```

## CLI Access

Agents can also use the `bridgectl` binary for local or containerized execution:

```bash
./bridgectl tool get
```

To call a tool directly, use the REST endpoint:

```bash
curl -X POST http://localhost:8080/api/tools/invoke \
  -H 'Content-Type: application/json' \
  -d '{"name": "erp.list_employees", "arguments": {}}'
```
