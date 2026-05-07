# ERP AI Middleware

Middleware for bridging legacy ERP systems with Agentic AI using the Model Context Protocol (MCP).

## Architecture

- **mock-erp/**: Temporary Python FastAPI service simulating legacy ERP modules (Finance, HR, Inventory).
- **services/erpbridge-server/**: Go-based MCP Server (HTTP/SSE) + ERP Connector.
- **tools/bridgectl/**: Go CLI for developers and AI agents to manage APIs and tools.
- **internal/**: Shared Go libraries for configuration, protocol handling, and I/O.

## Packages

| Package | Type | Binary Name | Description |
| :--- | :--- | :--- | :--- |
| **ERPBridge Server** | Service | `erpbridge-server` | The core MCP Server. Handles ERP connections, resilience, and semantic caching. |
| **bridgectl** | CLI | `bridgectl` | Developer tool for environment management, schema validation, and real-time monitoring. |

## Key Differences

| Feature | ERPBridge Server | bridgectl |
| :--- | :--- | :--- |
| **Primary Role** | Runtime execution and protocol bridging. | Development, debugging, and management. |
| **Connectivity** | Connects to Redis and Legacy ERP APIs. | Connects to ERPBridge Server API. |
| **Lifecycle** | Long-running daemon (Docker/Kubernetes). | Short-lived command execution. |
| **Interface** | SSE (for agents) / HTTP (for metrics/CLI). | Standard Output (Table/JSON/YAML). |
| **State** | Manages semantic cache and circuit breakers. | Stateless; reads configuration from `~/.erpbridge.yaml`. |

## Stack
- **Go**: 1.26.2
- **Python**: 3.11+
- **MCP Library**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **Database**: Redis (for Semantic Caching)
- **Monitoring**: Prometheus (Metrics)
- **Resilience**: Sony/GoBreaker & Avast/Retry-Go

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
   uv run main.py
   ```

2. **ERPBridge Server**:
   ```bash
   go run services/erpbridge-server/main.go
   ```

3. **bridgectl**:
   ```bash
   go build -o bridgectl tools/bridgectl/main.go
   ./bridgectl --help
   ```

## Complete Guide

### 1. Overview
ERPBridge acts as a translation layer between legacy ERP systems and modern AI agents. It exposes ERP functionality as MCP Tools, Resources, and Prompts, allowing LLMs to interact with complex business data through a standardized interface.

### 2. Core Components

#### ERPBridge Server
The ERPBridge Server service is an MCP Server implemented in Go. It:
- **Discovers Tools**: Maps ERP API endpoints to MCP definitions with **Hot Reloading** support.
- **Resilience**: Implements **Circuit Breaking** and **Intelligent Retries** to handle ERP instability.
- **Advanced MCP**: Supports Tools (actions), **Resources** (read-only data), and **Prompts** (workflows).
- **Semantic Caching**: Reduces ERP load using vector similarity matching.

#### bridgectl
The developer CLI for managing the ecosystem:
- **`bridgectl doc`**: Generate comprehensive Markdown documentation for the CLI.
- `bridgectl tool validate`: Pre-validate schemas or OpenAPI specs.
- `bridgectl tool list/invoke`: List and test MCP tools.
- `bridgectl api`: Explore raw ERP endpoints.
- `bridgectl log`: Stream real-time middleware logs.
- `bridgectl cache`: Manage and monitor semantic cache performance.

### 3. Resilience & Reliability
To protect against legacy system failures, the connector uses:
- **Circuit Breaker**: Automatically trips (opens) if the ERP failure rate exceeds 60%, preventing cascading failures.
- **Exponential Backoff**: Automatically retries transient errors (5xx, 429) up to 3 times with increasing delays.

### 4. Observability
- **Prometheus Metrics**: High-resolution metrics for latency, request counts, and error rates are available at `:8080/metrics`.
- **Cache Dashboard**: Real-time stats on exact vs. semantic hits are exposed via `bridgectl cache stats` or `/api/cache/stats`.
- **Structured Logging**: All requests carry a unique `request_id` for end-to-end tracing.

### 5. CLI Documentation
Comprehensive documentation for all `bridgectl` commands is available in the [docs/cli](./docs/cli) directory. You can regenerate this documentation at any time by running:
```bash
go run tools/bridgectl/main.go doc
```

### 6. Semantic Caching
ERPBridge features a two-layer caching strategy:
1.  **Exact Match**: Keyed by a hash of tool arguments.
2.  **Semantic Fallback**: Uses vector embeddings to find similar previous requests based on a similarity threshold (default: 0.95).

### 7. Connectivity & AI Agent Integration
ERPBridge supports multiple transport protocols for different client types:
- **Postman & Web**: Streamable HTTP at `http://localhost:8080/mcp/`.
- **AI Agents (Claude/Cursor)**: SSE at `http://localhost:8080/mcp/sse`.

See the comprehensive [Connectivity & Transport Guide](./docs/connectivity.md) and [AGENTS.md](./AGENTS.md) for detailed integration patterns.

### 8. Development Flow
1.  **Schema Hot-Reload**: Modify any JSON schema in `schemas/`; the middleware reloads it instantly.
2.  **Validation**: Use `bridgectl tool validate schemas/finance/invoices.json` before deploying.

## License
MIT
