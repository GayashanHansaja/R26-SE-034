# Docker Deployment Guide

This guide covers how to deploy and manage the ERPBridge ecosystem using Docker and Docker Compose.

## 1. Quick Start

Ensure you have [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) installed.

```bash
# Clone the repository
git clone https://github.com/nmdra/ERPBridge.git
cd ERPBridge

# Start the full stack
docker compose up -d --build
```

The stack includes:
- **ERPBridge Server** (`:8080`): The core MCP middleware.
- **Mock ERP** (`:8081`): Simulates legacy ERP endpoints.
- **Redis** (`:6379`): Provides semantic and exact matching cache.
- **Embedder** (`:8083`): HuggingFace text-embeddings-inference for semantic search.

## 2. Configuration

Environment variables can be configured in the `docker-compose.yml` file.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `BASE_URL` | Public URL of the MCP server (for SSE links). | `http://localhost:8080` |
| `ERP_BASE_URL` | Base URL of the underlying ERP system. | `http://mock-erp:8081` |
| `REDIS_URL` | URL for the Redis cache. | `redis://redis:6379` |
| `EMBEDDER_URL` | URL for the vector embedding service. | `http://embedder:8083` |

## 3. Schema Management

The `schemas/` directory is mounted as a volume in the `erpbridge-server` container:

```yaml
volumes:
  - ./schemas:/app/schemas
```

**Hot Reloading:** You can generate or modify tool schemas locally using `bridgectl`, and the server inside the container will automatically detect and register them without a restart. The server recursively watches the `schemas/` directory, meaning changes in subdirectories (e.g., `schemas/erp/`, `schemas/test/`) are fully supported.

## 4. Using bridgectl with Docker

You can use the local `bridgectl` binary to interact with the server running in Docker.

1.  **Build bridgectl:**
    ```bash
    go build -o bridgectl tools/bridgectl/main.go
    ```

2.  **Verify Connection:**
    ```bash
    ./bridgectl tool list
    ```

3.  **Generate a new tool:**
    ```bash
    # This writes to ./schemas/erp/ which is seen by the container
    ./bridgectl tool generate --api mock-erp --openapi mock-erp/openapi.yaml
    ```

## 5. Logs & Monitoring

- **View Container Logs:**
  ```bash
  docker compose logs -f erpbridge-server
  ```
- **Live Stream Logs via CLI:**
  ```bash
  ./bridgectl log tail
  ```
- **Metrics:**
  Prometheus metrics are available at `http://localhost:8080/metrics`.

## 6. Troubleshooting

- **Connection Refused:** Ensure `ERP_BASE_URL` in `docker-compose.yml` uses the service name `http://mock-erp:8081` rather than `localhost`.
- **Cache Issues:** If semantic search is not working, check the `embedder` container logs to ensure the model is loaded correctly.
- **Schema Errors:** Use `./bridgectl tool validate schemas/path/to/tool.json` to debug invalid tool definitions.
