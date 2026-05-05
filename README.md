# ERP AI Middleware

Middleware for bridging legacy ERP systems with Agentic AI using the Model Context Protocol (MCP).

## Architecture

- **Mock ERP**: Simulates legacy ERP modules (Finance, HR, Inventory).
- **Middleware**: MCP Server (HTTP/SSE) + ERP Connector.
- **bridgectl**: CLI for developers and AI agents to manage APIs and tools.

## Getting Started

### Prerequisites
- Go 1.23+
- Docker & Docker Compose

### Running the Stack
```bash
docker compose up -d
```

### Using bridgectl
```bash
# Register an API
bridgectl api register --name "finance.invoices" --url "http://localhost:8081/api/v1/finance/invoices" --module finance --description "List invoices" --auth-key "finance-key-001"

# Generate MCP tool
bridgectl tool generate --api finance.invoices

# List tools
bridgectl tool list

# Invoke tool
bridgectl tool invoke finance.invoices '{"page":1}'
```

## AI Agent Integration

AI agents can connect to the middleware at `http://localhost:8080/mcp`.
See [AGENTS.md](./AGENTS.md) for details.
