# Bridgectl Troubleshooting Guide

This guide contains common errors and fixes for the `bridgectl` workflow.

## Step 2: Register the API Errors

| Error | Cause | Fix |
|---|---|---|
| `name already exists` | Duplicate `--name` | Ask the user for a unique name |
| `invalid URL format` | Malformed URL | Correct URL format and retry |
| `unknown auth-type` | Unsupported value | Use the auth syntax reference in SKILL.md |
| `permission denied` | Registry file locked | Ask user to check file permissions |

## Step 3: Test Connectivity Errors

| Symptom | Likely cause | Fix |
|---|---|---|
| `connection refused` | ERP server unreachable | Ask user to verify the ERP is running and accessible from the middleware host |
| `401 Unauthorized` | Wrong credential | Re-register with the correct `--auth-key` |
| `403 Forbidden` | IP allowlist or scope issue | Ask user to whitelist the middleware server IP in the ERP |
| `404 Not Found` | Wrong endpoint URL | Verify the URL with the user and re-register |
| `timeout` | Network or firewall | Ask user to check routing between middleware and ERP |

## Step 5: Apply & Verify Errors

| Error | Cause | Fix |
|---|---|---|
| `invalid schema: missing spec.description` | Required field empty | Fill the field and retry |
| `tool name conflict` | Tool already registered | Add `--force` to overwrite, or rename |
| `mcp server unreachable` | Wrong context URL | Run `bridgectl context list` and verify the active context |
| `YAML parse error` | Malformed YAML | Check indentation, then retry |
