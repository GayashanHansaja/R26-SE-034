# ERP AI Middleware

Middleware for bridging legacy ERP systems with Agentic AI using the Model Context Protocol (MCP).

## Architecture

- **mock-erp/**: Temporary Python FastAPI service simulating legacy ERP modules (Finance, HR, Inventory).
- **services/middleware/**: Go-based MCP Server (HTTP/SSE) + ERP Connector.
- **tools/bridgectl/**: Go CLI for developers and AI agents to manage APIs and tools.
- **internal/**: Shared Go libraries for configuration, protocol handling, and I/O.

## Stack
- **Go**: 1.26.2
- **Python**: 3.11+
- **MCP Library**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **YAML Library**: [goccy/go-yaml](https://github.com/goccy/go-yaml)

## Getting Started

### Prerequisites
- Go 1.26.2+
- Python 3.11+
- Docker & Docker Compose

### Running the Full Stack
```bash
docker compose up -d --build
```

### Manual Build & Run
1. **Mock ERP**:
   ```bash
   cd mock-erp
   python -m venv venv
   source venv/bin/activate
   pip install -r requirements.txt
   python main.py
   ```

2. **Middleware**:
   ```bash
   go run services/middleware/main.go
   ```

3. **bridgectl**:
   ```bash
   go build -o bridgectl tools/bridgectl/main.go
   ./bridgectl --help
   ```

## AI Agent Integration
AI agents can connect to the middleware via SSE at `http://localhost:8080/mcp/sse`.
See [AGENTS.md](./AGENTS.md) for details.
