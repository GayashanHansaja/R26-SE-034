# ERPBridge Onboarding Guide: New API

This guide outlines the standard workflow for onboarding a new ERP system into ERPBridge using the `bridgectl` CLI.

## Prerequisites
- **Docker & Docker Compose**: To run the ERPBridge server and Mock ERP.
- **Go 1.22+**: To build the `bridgectl` CLI (if not already built).

---

## 1. Environment Setup
The ERPBridge server must be running to receive registrations and tool applications.

```bash
# Rebuild and start services
docker-compose up --build -d

# Verify services are healthy
docker-compose ps
```
*The server will be available at `http://localhost:8080`.*

## 2. Prepare the CLI
If `bridgectl` is not present in your root directory, build it:

```bash
make build
# Or manually: go build -o bridgectl ./tools/bridgectl/main.go
```

## 3. Register the ERP API
Registration defines the connection details for the middleware.

```bash
./bridgectl api register \
  --name erp \
  --url http://localhost:8081 \
  --module erp \
  --description "Internal Mock ERP for testing"
```
- `--name`: Unique identifier for the API.
- `--url`: The base URL of the ERP service.
- `--module`: Logical grouping (e.g., finance, hr, erp).

## 4. Generate MCP Tool Schemas
Generate declarative JSON schemas from the ERP's OpenAPI specification.

```bash
./bridgectl tool generate --api erp --openapi mock-erp/openapi.yaml
```
- This command populates the `schemas/erp/` directory with individual tool definitions.

## 5. Apply Tools to the Registry
Upload the generated schemas to the ERPBridge server. You can apply a single file or an entire directory.

```bash
# Apply all tools in a directory
./bridgectl tool apply -f schemas/erp/

# Or apply a single tool
./bridgectl tool apply -f schemas/erp/list_employees.json
```

## 6. Verification
Confirm that the tools are registered and in a `READY` state.

```bash
./bridgectl tool get
```

## 7. Deleting Tools
If a tool is no longer needed or needs to be temporarily disabled, you can delete it from the registry.

```bash
./bridgectl tool delete [tool_name] [version]
# Example:
./bridgectl tool delete list_items 1.0.0
```

> **Note**: Deleted tools are not immediately purged; they transition to a `HIDDEN` status. This removes them from the MCP advertisement list while keeping them in the registry for audit/reconciliation. To restore a hidden tool, simply re-run the `apply` command for that tool's schema.

---

## Troubleshooting Guide

### 1. Server Connection Issues
**Error**: `apply failed: Get "http://localhost:8080/...": dial tcp 127.0.0.1:8080: connect: connection refused`
- **Cause**: The `erpbridge-server` is not running or the CLI is looking at the wrong context.
- **Fix**: 
  - Check Docker: `docker-compose ps`.
  - Check context: `./bridgectl context list`. If `local` is missing or incorrect, set it:
    ```bash
    ./bridgectl context set local --server http://localhost:8080
    ```

### 2. Missing Registration Flags
**Error**: `required flag(s) "description" not set`
- **Cause**: The `api register` command requires a description to help LLMs understand the API's purpose.
- **Fix**: Always include `--description "..."` in the registration command.

### 3. OpenAPI Path Errors
**Error**: `failed to load OpenAPI spec: open ...: no such file or directory`
- **Cause**: The path provided to `--openapi` is incorrect relative to your current directory.
- **Fix**: Verify the path with `ls mock-erp/openapi.yaml`.

### 5. Redis Connectivity
**Symptom**: Tools are applied, but calls to them fail with `internal server error`.
- **Fix**: Check `erpbridge-server` logs for Redis errors:
  ```bash
  docker-compose logs erpbridge-server
  ```
  Ensure the `redis` container is healthy.
