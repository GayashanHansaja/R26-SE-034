---
name: bridgectl-add-api
description: Autonomously register a new ERP API endpoint and expose it as a callable MCP tool via bridgectl. Use for onboarding new APIs, generating tool schemas, and verifying ERP connectivity.
license: MIT
compatibility: "Requires bridgectl CLI, Go 1.22+, and network access to the target ERP and ERPBridge Management Server."
metadata:
  version: "2.0.0"
  author: "ERPBridge Team"
allowed-tools: "run_shell_command read_file write_file replace"
---

This skill guides an AI agent through the deterministic workflow of registering an ERP API and exposing it as an MCP tool. 

> [!IMPORTANT]
> Always run steps in order. Never skip Step 3 (Test Connectivity). Never apply a schema without enhancing its metadata.

## Architecture Overview

| URL Role | Config Key | Purpose |
|---|---|---|
| Management Server | `server` | Control plane for `tool apply`, `list`, `log`. |
| MCP Runtime | `mcp-server` | Entry point for AI agents via `tool invoke`. |
| ERP System | `erp-base` | The legacy system being wrapped. |

For troubleshooting common setup or runtime errors, see [Troubleshooting Guide](references/TROUBLESHOOTING.md).

## Workflow

### Step 0: Pre-flight
Verify the environment and active context:
```bash
bridgectl version && bridgectl context list && bridgectl doc
```

### Step 1: Collect Information
Gather all required fields: Name (kebab-case), URL, HTTP Method, Module, Description, and Auth details. 
*Note: Refer to CLI docs in `docs/cli/` for specific flag details.*

### Step 2: Register the API
```bash
bridgectl api register \
  --name <NAME> \
  --url "<URL>" \
  --method <METHOD> \
  --module <MODULE> \
  --description "<DESC>" \
  --auth-type <AUTH_TYPE> \
  --auth-key "<CREDENTIAL>"
```

### Step 3: Test Connectivity (Mandatory)
```bash
bridgectl api test <NAME>
```
**Stop if this fails.** Refer to [Troubleshooting: Step 3](references/TROUBLESHOOTING.md#step-3-test-connectivity-errors).

### Step 4: Generate & Enhance Schema
Generate the declarative YAML:
```bash
bridgectl tool generate --api <NAME> -o yaml > <NAME>.yaml
```

**Required Enhancement:** Use the [MCP Tool Template](assets/mcp-tool.yaml) as a reference. You MUST fill in:
- `spec.description.short`: One concise sentence.
- `spec.description.whenToUse`: Natural language trigger conditions.
- `spec.description.examples`: 2-3 user phrases that should trigger this tool.

### Step 5: Apply & Verify
```bash
bridgectl tool apply -f <NAME>.yaml
```
Verify the tool is live:
```bash
bridgectl tool list && bridgectl tool describe <NAME>
```

---

## Resources
- [Quick Command Reference](references/COMMANDS.md)
- [Troubleshooting & Error Codes](references/TROUBLESHOOTING.md)
- [MCP Tool YAML Template](assets/mcp-tool.yaml)
