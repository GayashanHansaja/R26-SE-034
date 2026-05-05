# AI Agent Integration Guide

This middleware exposes ERP functionality via the **Model Context Protocol (MCP)**.

## MCP Endpoints

- **List Tools**: `GET /mcp/tools/list`
- **Invoke Tool**: `POST /mcp/tools/call`

## Example Usage (cURL)

### List Available Tools
```bash
curl http://localhost:8080/mcp/tools/list
```

### Call a Tool
```bash
curl -X POST http://localhost:8080/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "finance.invoices",
    "arguments": {
      "page": 1
    }
  }'
```

## Integration Patterns

1. **Discovery**: AI agents should first call `/mcp/tools/list` to understand the available capabilities and their required input schemas.
2. **Execution**: Use `/mcp/tools/call` to perform actions. The middleware handles authentication to the legacy ERP.
3. **CLI Access**: Agents can also use the `bridgectl` binary with the `-o json` flag for local or containerized execution.
