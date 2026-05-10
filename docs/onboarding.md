# ERPBridge Onboarding Guide

> Get a new ERP system connected to ERPBridge in under 10 minutes using the `bridgectl` CLI.

---

## Before You Start

Make sure you have the following installed:

| Requirement | Purpose |
|---|---|
| Docker & Docker Compose | Runs the ERPBridge server and Mock ERP |
| Go 1.22+ | Needed to build `bridgectl` (if not pre-built) |

---

## Step 1 — Start the ERPBridge Server

Start all required services with Docker Compose:

```bash
docker-compose up --build -d
```

Confirm everything is running:

```bash
docker-compose ps
```

The ERPBridge server will be available at **`http://localhost:8080`**.

---

## Step 2 — Build the CLI

If you don't already have `bridgectl` in your project root, build it now:

```bash
make build
```

Or build it manually:

```bash
go build -o bridgectl ./tools/bridgectl/main.go
```

---

## Step 3 — Register Your ERP API

Tell ERPBridge how to connect to your ERP system:

```bash
./bridgectl api register \
  --name erp \
  --url http://localhost:8081 \
  --module erp \
  --description "Internal Mock ERP for testing"
```

**What each flag does:**

| Flag | Description |
|---|---|
| `--name` | Unique identifier for this API |
| `--url` | Base URL of your ERP service |
| `--module` | Logical grouping (e.g., `finance`, `hr`, `erp`) |
| `--description` | Human-readable description — required by the system |

> **Tip:** The `--description` flag is mandatory. It helps the LLM layer understand the API's purpose.

---

## Step 4 — Generate Tool Schemas

Convert your ERP's OpenAPI spec into MCP-compatible JSON tool schemas:

```bash
./bridgectl tool generate --api erp --openapi mock-erp/openapi.yaml
```

This populates the `schemas/erp/` directory with one JSON file per tool definition.

> **Check the path first** if you're unsure: `ls mock-erp/openapi.yaml`

---

## Step 5 — Apply Tools to the Registry

Upload your generated schemas to the ERPBridge server.

**Apply all tools at once:**

```bash
./bridgectl tool apply -f schemas/erp/
```

**Or apply a single tool:**

```bash
./bridgectl tool apply -f schemas/erp/list_employees.json
```

---

## Step 6 — Verify Everything Is Working

Confirm your tools are registered and in a `READY` state:

```bash
./bridgectl tool get
```

You should see your tools listed with a `READY` status. If any show a different status, see the Troubleshooting section below.

---

## Managing Tools

### Deleting a Tool

Remove a tool from the active registry when it's no longer needed:

```bash
./bridgectl tool delete [tool_name] [version]

# Example:
./bridgectl tool delete list_items 1.0.0
```

> **Note:** Deleted tools aren't permanently removed. They transition to `HIDDEN` status — invisible to MCP clients, but still in the registry for audit purposes.
>
> To restore a hidden tool, simply re-apply its schema:
> ```bash
> ./bridgectl tool apply -f schemas/erp/list_items.json
> ```

---

## Troubleshooting

### Connection refused when running CLI commands

**Error:**
```
apply failed: Get "http://localhost:8080/...": dial tcp 127.0.0.1:8080: connect: connection refused
```

**Cause:** The ERPBridge server isn't running, or the CLI is pointed at the wrong address.

**Fix:**
1. Check that Docker services are up:
   ```bash
   docker-compose ps
   ```
2. Check your CLI context:
   ```bash
   ./bridgectl context list
   ```
3. If `local` is missing or wrong, set it:
   ```bash
   ./bridgectl context set local --server http://localhost:8080
   ```

---

### Registration fails with a missing flag error

**Error:**
```
required flag(s) "description" not set
```

**Cause:** The `--description` flag is required on `api register`.

**Fix:** Always include it:
```bash
./bridgectl api register \
  --name erp \
  --url http://localhost:8081 \
  --module erp \
  --description "Internal Mock ERP for testing"
```

---

### OpenAPI spec not found

**Error:**
```
failed to load OpenAPI spec: open ...: no such file or directory
```

**Cause:** The path passed to `--openapi` doesn't match the actual file location.

**Fix:** Verify the file exists from your current directory:
```bash
ls mock-erp/openapi.yaml
```

Then re-run the generate command with the correct path.

---

### Tools are applied but calls return `internal server error`

**Cause:** The ERPBridge server likely can't reach Redis.

**Fix:** Check the server logs for Redis-related errors:
```bash
docker-compose logs erpbridge-server
```

Ensure the `redis` container is healthy in `docker-compose ps`. Restart if needed:
```bash
docker-compose restart redis
```

---

## Quick Reference

```bash
# Start services
docker-compose up --build -d

# Build CLI
make build

# Register API
./bridgectl api register --name erp --url http://localhost:8081 --module erp --description "..."

# Generate schemas from OpenAPI spec
./bridgectl tool generate --api erp --openapi mock-erp/openapi.yaml

# Apply all tools
./bridgectl tool apply -f schemas/erp/

# Verify tools are READY
./bridgectl tool get

# Delete a tool (sets to HIDDEN)
./bridgectl tool delete [tool_name] [version]
```