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
- **Database**: Redis (for Semantic Caching)

## Getting Started

### Prerequisites
- Go 1.26.2+
- Python 3.11+
- Docker & Docker Compose
- Redis (with RediSearch module)

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

## Complete Guide

### 1. Overview
ERPBridge acts as a translation layer between legacy ERP systems and modern AI agents. It exposes ERP functionality as MCP Tools, allowing LLMs to interact with complex business data through a standardized interface.

### 2. Core Components

#### Middleware
The Middleware service is an MCP Server implemented in Go. It:
- **Discovers Tools**: Maps ERP API endpoints to MCP tool definitions.
- **Handles Requests**: Proxies tool calls from AI agents to the ERP.
- **Manages Context**: Injects authentication and maintains session state.
- **Semantic Caching**: Reduces ERP load by caching responses with vector similarity.

#### Mock ERP
A Python-based service providing simulated ERP modules:
- **Finance**: Invoices, Payments, Ledger.
- **HR**: Employee records, Payroll, Attendance.
- **Inventory**: Stock levels, Warehouses, Shipments.

#### bridgectl
The developer CLI for managing the ecosystem:
- `bridgectl context`: Switch between local, dev, and prod environments.
- `bridgectl api`: Explore and test raw ERP API endpoints.
- `bridgectl tool`: List and test MCP tools exposed by the middleware.
- `bridgectl log`: Stream real-time logs from the middleware.
- `bridgectl cache`: Manage the semantic cache (stats, flush).

### 3. Semantic Caching
ERPBridge features a unique two-layer caching strategy powered by Redis:
1.  **Exact Match**: Keyed by a hash of tool arguments for 100% precision.
2.  **Semantic Fallback**: Uses vector embeddings to find "similar" previous requests. If an agent asks for "Latest invoices for Q1" and later asks "Most recent Q1 billing data", the middleware can return the cached result if the similarity score exceeds the configured threshold.

**Configuration**:
```yaml
cache:
  enabled: true
  ttlSeconds: 3600
  semanticThreshold: 0.95
  isReadOnly: false # Role-isolated caching
```

### 4. Configuration
`bridgectl` and the Middleware use a shared configuration model. By default, it looks for `~/.bridgectl/config.yaml`.

**Environment Variables**:
- `BRIDGE_SERVER`: Middleware URL.
- `BRIDGE_ERP_BASE`: Backend ERP URL.
- `BRIDGE_API_KEY`: API Key for ERP authentication.

### 5. AI Agent Integration
AI agents can connect to the middleware via SSE at `http://localhost:8080/mcp/sse`.
The middleware provides a `listTools` capability that agents use to discover available ERP operations.

See [AGENTS.md](./AGENTS.md) for detailed integration patterns.

### 6. Development Flow
To add a new ERP capability:
1.  **Mock ERP**: Add a new route in `mock-erp/routers/`.
2.  **Middleware**: Update the tool registry to include the new endpoint.
3.  **Verification**: Use `bridgectl tool list` and `bridgectl tool call` to verify.

## License
MIT
