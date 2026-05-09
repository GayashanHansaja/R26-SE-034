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
- **Redis** (`:6379`): Provides exact matching cache.

## 2. Configuration

Environment variables can be configured in the `docker-compose.yml` file.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `BASE_URL` | Public URL of the MCP server (for SSE links). | `http://localhost:8080` |
| `ERP_BASE_URL` | Base URL of the underlying ERP system. | `http://mock-erp:8081` |
| `REDIS_URL` | URL for the Redis cache. | `redis://redis:6379` |

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

## 6. Connecting MCP Clients

ERPBridge can be connected to any MCP-compatible client using either the **Stdio** or **HTTP (SSE)** transport.

### Claude Desktop (Stdio)

Claude Desktop typically interacts with MCP servers via standard I/O inside a Docker container.

1.  **Locate Configuration:**
    - **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
    - **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

2.  **Add ERPBridge Server:**
    Add the following to the `mcpServers` section. This command runs the server in "stdio" mode inside a container.

    ```json
    {
      "mcpServers": {
        "erpbridge": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-v", "/absolute/path/to/ERPBridge/schemas:/app/schemas",
            "erpbridge-server:latest",
            "--stdio"
          ]
        }
      }
    }
    ```
    *Note: Replace `/absolute/path/to/ERPBridge/schemas` with your actual local path to ensure the container can see your tool definitions.*

3.  **Restart Claude:** Fully quit and restart Claude Desktop. Look for the tool icon in the chat input.

### Cursor (HTTP / SSE)

Cursor supports connecting to remote MCP servers via HTTP. This is the recommended method when your ERPBridge stack is already running via `docker compose up`.

1.  **Ensure Server is Running:**
    Verify your stack is up and the server is reachable at `http://localhost:8080`.

2.  **Configure Cursor:**
    - Open Cursor **Settings** (`Cmd+,` or `Ctrl+,`).
    - Navigate to **Features** > **MCP**.
    - Click **+ Add New MCP Server**.
    - **Name:** `ERPBridge`
    - **Type:** `sse`
    - **URL:** `http://localhost:8080/mcp/sse` (Note: The server appends `/mcp/` to the base URL).

3.  **Verify:**
    Once added, you should see a green status indicator. You can now use ERP tools in Cursor Chat or Composer.

## 7. Troubleshooting

- **Connection Refused:** Ensure `ERP_BASE_URL` in `docker-compose.yml` uses the service name `http://mock-erp:8081` rather than `localhost`.
- **Claude Stdio Timeout:** If Claude fails to connect, try building the server binary first and running it directly to ensure there are no startup errors (e.g., missing dependencies).
- **Schema Errors:** Use `./bridgectl tool validate schemas/path/to/tool.json` to debug invalid tool definitions.

