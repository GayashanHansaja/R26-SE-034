# MCP Client Guide (Python & TypeScript)

> **Who is this guide for?** This guide is written for developers who are new to MCP (Model Context Protocol). It walks you through building clients that connect to an ERPBridge server using either **Streamable HTTP** or **Stdio** transport — with complete, copy-paste-ready examples in both Python and TypeScript.

---

## Table of Contents

1. [What is MCP?](#1-what-is-mcp)
2. [Prerequisites](#2-prerequisites)
3. [Choosing a Transport](#3-choosing-a-transport)
4. [How Streamable HTTP Works](#4-how-streamable-http-works)
5. [Python Client — Streamable HTTP](#5-python-client--streamable-http)
6. [Python Client — Stdio](#6-python-client--stdio)
7. [TypeScript Client — Streamable HTTP](#7-typescript-client--streamable-http)
8. [TypeScript Client — Stdio](#8-typescript-client--stdio)
9. [Error Handling](#9-error-handling)
10. [Troubleshooting](#10-troubleshooting)
11. [Logging & Redaction](#11-logging--redaction)
12. [Quick Reference Cheat Sheet](#12-quick-reference-cheat-sheet)

---

## 1. What is MCP?

**MCP (Model Context Protocol)** is a standard protocol that lets clients call tools exposed by a server — similar to calling a REST API, but designed for AI and automation workflows.

In this guide, the server is **ERPBridge**, which exposes business tools (like listing invoices) over MCP. Your job as a client is to:

1. **Connect** to the server and start a session.
2. **Discover** which tools are available.
3. **Call** those tools and receive results.

All communication uses **JSON-RPC 2.0** — a simple, human-readable message format.

---

## 2. Prerequisites

### 2.1 Start the ERPBridge Server

Before writing any client code, you need a running ERPBridge server to connect to.

**Option A — Docker (recommended for beginners):**
```bash
docker compose up -d --build
```

**Option B — Run the HTTP server locally:**
```bash
go run services/erpbridge-server/main.go
```

**Option C — Run the Stdio server locally:**
```bash
go run services/erpbridge-server/main.go --stdio
```

> **Tip:** If you're unsure which option to choose, use Docker. It handles all dependencies automatically.

### 2.2 Verify the Server is Running

Once started, the server listens at:

```
http://localhost:8080/mcp/
```

You can change the port with the `MCP_PORT` environment variable or the full address with `BASE_URL`.

**Quick check — confirm the server responds:**
```bash
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/mcp/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}'
```

You should see `200`.

### 2.3 Install Language Dependencies

**Python:**
```bash
pip install requests
```

**TypeScript / Node.js:**
```bash
npm install node-fetch   # only needed for Node.js < 18; built-in fetch works in Node 18+
```

---

## 3. Choosing a Transport

MCP supports two ways to communicate with the server:

| Transport | How it works | Best for |
|---|---|---|
| **Streamable HTTP** | Send JSON-RPC messages over regular HTTP POST requests. Optionally receive server-sent events (SSE) for real-time notifications. | Web apps, services, API testing tools like Postman |
| **Stdio** | Spawn the server as a child process and communicate over its standard input/output (stdin/stdout). | Local scripts, CLI tools, desktop integrations |

**Not sure which to pick?** Start with **Streamable HTTP** — it's easier to debug since you can inspect requests in a browser or Postman.

---

## 4. How Streamable HTTP Works

Every message you send follows the **JSON-RPC 2.0** format and is sent as a `POST` to `/mcp/`. Here's the typical flow:

```
Client                          ERPBridge Server
  |                                    |
  |--- POST /mcp/ (initialize) ------->|
  |<-- 200 OK + Mcp-Session-Id --------|   ← Save this header!
  |                                    |
  |--- POST /mcp/ (tools/list) ------->|   ← Include session ID
  |<-- List of available tools --------|
  |                                    |
  |--- POST /mcp/ (tools/call) ------->|   ← Call a specific tool
  |<-- Tool result ------------------- |
  |                                    |
  |--- GET /mcp/ (SSE, optional) ----->|   ← Subscribe to notifications
  |<-- Server-sent events (ongoing) ---|
```

### 4.1 Step 1 — Initialize the Session

Send an `initialize` request first. The server replies with a `Mcp-Session-Id` header that you must include in all subsequent requests.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": { "name": "my-client", "version": "0.1.0" }
  }
}
```

> **Why is the session ID important?** The server uses it to associate your requests with your session. Without it, subsequent calls will fail.

### 4.2 Step 2 — List Available Tools

```json
{ "jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {} }
```

This returns a list of tool names and their descriptions. Always call this before `tools/call` to confirm the exact tool name.

### 4.3 Step 3 — Call a Tool

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "finance.list_invoices_api_v1_finance_invoices_get",
    "arguments": {}
  }
}
```

> **Important:** The tool name must match exactly what `tools/list` returned — including underscores and casing.

### 4.4 Step 4 (Optional) — Subscribe to Notifications

Open a `GET` request with `Accept: text/event-stream` to receive real-time server-sent events:

```
GET /mcp/
Accept: text/event-stream
Mcp-Session-Id: <your-session-id>
```

Notification types you may receive:

| Notification | Meaning |
|---|---|
| `notifications/progress` | A long-running tool is reporting progress |
| `notifications/message` | A log message from the server |
| `notifications/alert` | An important alert or warning |
| `notifications/tools/list_changed` | Available tools have changed |

---

## 5. Python Client — Streamable HTTP

### 5.1 Minimal Working Example

This example connects to ERPBridge, lists available tools, and calls one.

```python
import json
import requests

BASE_URL = "http://localhost:8080/mcp/"

# --- Step 1: Initialize the session ---
init_payload = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "python-client", "version": "0.1.0"},
    },
}

session = requests.Session()
init_resp = session.post(BASE_URL, json=init_payload)
init_resp.raise_for_status()

# Save the session ID — required for all future requests
session_id = init_resp.headers.get("Mcp-Session-Id")
if not session_id:
    raise RuntimeError("Server did not return Mcp-Session-Id. Did initialize succeed?")

headers = {"Mcp-Session-Id": session_id}
print(f"Session started: {session_id}")

# --- Step 2: List available tools ---
tools_resp = session.post(
    BASE_URL,
    headers=headers,
    json={"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}},
)
tools_resp.raise_for_status()
tools = tools_resp.json()

print("Available tools:")
for tool in tools.get("result", {}).get("tools", []):
    print(f"  - {tool['name']}: {tool.get('description', 'No description')}")

# --- Step 3: Call a tool ---
call_resp = session.post(
    BASE_URL,
    headers=headers,
    json={
        "jsonrpc": "2.0",
        "id": 3,
        "method": "tools/call",
        "params": {
            "name": "finance.list_invoices_api_v1_finance_invoices_get",
            "arguments": {},
        },
    },
)
call_resp.raise_for_status()
result = call_resp.json()

print("\nTool result:")
print(json.dumps(result, indent=2))
```

### 5.2 Calling Tools With Arguments

Some tools require arguments. Pass them as a dictionary in the `arguments` field:

```python
call_resp = session.post(
    BASE_URL,
    headers=headers,
    json={
        "jsonrpc": "2.0",
        "id": 4,
        "method": "tools/call",
        "params": {
            "name": "finance.get_invoice",
            "arguments": {
                "invoice_id": "INV-2024-001",
                "include_line_items": True,
            },
        },
    },
)
result = call_resp.json()
```

### 5.3 Receiving Notifications (SSE)

```python
import sseclient  # pip install sseclient-py

with session.get(
    BASE_URL,
    headers={
        "Accept": "text/event-stream",
        "Mcp-Session-Id": session_id,
    },
    stream=True,
) as resp:
    client = sseclient.SSEClient(resp)
    for event in client.events():
        notification = json.loads(event.data)
        print(f"Notification [{notification.get('method')}]: {notification}")
```

---

## 6. Python Client — Stdio

Use this approach when you want to run the ERPBridge server as a subprocess and communicate directly through its standard input/output.

### 6.1 Minimal Working Example

```python
import json
import subprocess

# --- Start the server as a child process ---
proc = subprocess.Popen(
    ["go", "run", "services/erpbridge-server/main.go", "--stdio"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,  # Capture stderr so it doesn't pollute your output
    text=True,
)

def send(message: dict):
    """Send a JSON-RPC message to the server via stdin."""
    proc.stdin.write(json.dumps(message) + "\n")
    proc.stdin.flush()

def receive() -> dict:
    """Read one JSON-RPC response from the server via stdout."""
    line = proc.stdout.readline()
    if not line:
        raise RuntimeError("Server process closed unexpectedly.")
    return json.loads(line)

# --- Step 1: Initialize ---
send({
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "python-stdio", "version": "0.1.0"},
    },
})
init_response = receive()
print("Initialized:", json.dumps(init_response, indent=2))

# --- Step 2: List tools ---
send({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
tools_response = receive()
print("Tools:", json.dumps(tools_response, indent=2))

# --- Step 3: Call a tool ---
send({
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
        "name": "finance.list_invoices_api_v1_finance_invoices_get",
        "arguments": {},
    },
})
result = receive()
print("Result:", json.dumps(result, indent=2))

# --- Always clean up ---
proc.stdin.close()
proc.wait()
```

> **Note:** Stdio mode doesn't use session IDs — the session is tied to the lifetime of the subprocess.

---

## 7. TypeScript Client — Streamable HTTP

### 7.1 Minimal Working Example

```typescript
const BASE_URL = "http://localhost:8080/mcp/";

async function main() {
  // --- Step 1: Initialize the session ---
  const initResp = await fetch(BASE_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 1,
      method: "initialize",
      params: {
        protocolVersion: "2024-11-05",
        capabilities: {},
        clientInfo: { name: "ts-client", version: "0.1.0" },
      },
    }),
  });

  if (!initResp.ok) {
    throw new Error(`Initialize failed: ${initResp.status} ${initResp.statusText}`);
  }

  // Save the session ID — required for all future requests
  const sessionId = initResp.headers.get("Mcp-Session-Id");
  if (!sessionId) {
    throw new Error("Server did not return Mcp-Session-Id. Did initialize succeed?");
  }

  const headers = {
    "Content-Type": "application/json",
    "Mcp-Session-Id": sessionId,
  };

  console.log(`Session started: ${sessionId}`);

  // --- Step 2: List available tools ---
  const toolsResp = await fetch(BASE_URL, {
    method: "POST",
    headers,
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 2,
      method: "tools/list",
      params: {},
    }),
  });

  const tools = await toolsResp.json();
  console.log("Available tools:", JSON.stringify(tools, null, 2));

  // --- Step 3: Call a tool ---
  const callResp = await fetch(BASE_URL, {
    method: "POST",
    headers,
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: 3,
      method: "tools/call",
      params: {
        name: "finance.list_invoices_api_v1_finance_invoices_get",
        arguments: {},
      },
    }),
  });

  const result = await callResp.json();
  console.log("Tool result:", JSON.stringify(result, null, 2));
}

main().catch(console.error);
```

### 7.2 Calling Tools With Arguments

```typescript
const callResp = await fetch(BASE_URL, {
  method: "POST",
  headers,
  body: JSON.stringify({
    jsonrpc: "2.0",
    id: 4,
    method: "tools/call",
    params: {
      name: "finance.get_invoice",
      arguments: {
        invoice_id: "INV-2024-001",
        include_line_items: true,
      },
    },
  }),
});
const result = await callResp.json();
```

### 7.3 Receiving Notifications (SSE)

```typescript
const streamResp = await fetch(BASE_URL, {
  headers: {
    Accept: "text/event-stream",
    "Mcp-Session-Id": sessionId,
  },
});

const reader = streamResp.body?.getReader();
const decoder = new TextDecoder();

if (reader) {
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;

    const text = decoder.decode(value);
    for (const line of text.split("\n")) {
      if (line.startsWith("data: ")) {
        const notification = JSON.parse(line.slice(6));
        console.log("Notification:", notification);
      }
    }
  }
}
```

---

## 8. TypeScript Client — Stdio

```typescript
import { spawn } from "node:child_process";
import * as readline from "node:readline";

// --- Start the server as a child process ---
const proc = spawn(
  "go",
  ["run", "services/erpbridge-server/main.go", "--stdio"],
  {
    stdio: ["pipe", "pipe", "inherit"], // inherit stderr so errors are visible
  }
);

// Use readline to read one line at a time from stdout
const rl = readline.createInterface({ input: proc.stdout });
const pendingReads: Array<(line: string) => void> = [];

rl.on("line", (line) => {
  const resolve = pendingReads.shift();
  if (resolve) resolve(line);
});

function send(message: object): void {
  proc.stdin.write(JSON.stringify(message) + "\n");
}

function receive(): Promise<object> {
  return new Promise((resolve) => {
    pendingReads.push((line) => resolve(JSON.parse(line)));
  });
}

async function main() {
  // --- Step 1: Initialize ---
  send({
    jsonrpc: "2.0",
    id: 1,
    method: "initialize",
    params: {
      protocolVersion: "2024-11-05",
      capabilities: {},
      clientInfo: { name: "ts-stdio", version: "0.1.0" },
    },
  });

  const initResponse = await receive();
  console.log("Initialized:", JSON.stringify(initResponse, null, 2));

  // --- Step 2: List tools ---
  send({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} });
  const toolsResponse = await receive();
  console.log("Tools:", JSON.stringify(toolsResponse, null, 2));

  // --- Step 3: Call a tool ---
  send({
    jsonrpc: "2.0",
    id: 3,
    method: "tools/call",
    params: {
      name: "finance.list_invoices_api_v1_finance_invoices_get",
      arguments: {},
    },
  });

  const result = await receive();
  console.log("Result:", JSON.stringify(result, null, 2));

  // --- Always clean up ---
  proc.stdin.end();
}

main().catch(console.error);
```

---

## 9. Error Handling

JSON-RPC errors are returned inside the response body, not as HTTP error codes. Always check the response for an `error` field.

### 9.1 What a JSON-RPC Error Looks Like

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": "No tool named 'finance.bad_tool_name'"
  }
}
```

### 9.2 Python — Checking for Errors

```python
response = call_resp.json()

if "error" in response:
    err = response["error"]
    print(f"Error {err['code']}: {err['message']}")
    if "data" in err:
        print(f"Details: {err['data']}")
else:
    result = response.get("result")
    print("Success:", json.dumps(result, indent=2))
```

### 9.3 TypeScript — Checking for Errors

```typescript
const response = await callResp.json();

if ("error" in response) {
  const { code, message, data } = response.error;
  console.error(`Error ${code}: ${message}`);
  if (data) console.error("Details:", data);
} else {
  console.log("Success:", JSON.stringify(response.result, null, 2));
}
```

### 9.4 Common JSON-RPC Error Codes

| Code | Meaning | Fix |
|---|---|---|
| `-32700` | Parse error | Your request body contains invalid JSON |
| `-32600` | Invalid request | Missing `jsonrpc`, `id`, or `method` fields |
| `-32601` | Method not found | Tool name doesn't exist — check with `tools/list` |
| `-32602` | Invalid params | Wrong or missing arguments for the tool |
| `-32603` | Internal error | Server-side error — check server logs |

---

## 10. Troubleshooting

### ❌ `Mcp-Session-Id` is missing from the response

**Cause:** The session wasn't initialized properly.

**Fix:**
- Make sure your very first request uses `"method": "initialize"`.
- Make sure you're sending to `POST /mcp/` — not `GET`, not `/mcp` without the trailing slash.

---

### ❌ `404 Not Found` or connection refused

**Cause:** The server isn't running or the URL is wrong.

**Fix:**
- Confirm the server started successfully (check Docker logs or terminal output).
- Verify the port: `curl http://localhost:8080/mcp/`
- If you changed the port, update `BASE_URL` in your client to match.

---

### ❌ Tool not found / wrong tool name

**Cause:** Tool names are case-sensitive and can be long.

**Fix:**
- Always call `tools/list` first and copy the `name` field exactly.
- Don't guess the tool name from the description.

```python
# Print all tool names to copy from
for tool in tools["result"]["tools"]:
    print(tool["name"])
```

---

### ❌ No notifications received from SSE stream

**Cause:** Notifications are optional and only sent for specific events.

**Fix:** This is usually expected behaviour. Notifications only appear when:
- A tool reports progress
- The server logs a message
- The tool list changes

You don't need SSE for basic tool calls to work.

---

### ❌ Stdio client hangs / no response

**Cause:** The process isn't writing to stdout, or you're not reading line-by-line.

**Fix:**
- Make sure you're reading one full line at a time (responses are newline-delimited).
- Check stderr for server startup errors.
- Confirm Go is installed: `go version`

---

## 11. Logging & Redaction

ERPBridge implements standard MCP logging via `notifications/message`. This allows servers to push real-time logs to clients, which is useful for debugging tool execution.

### 11.1 Setting the Log Level
Clients can control server-side verbosity by calling `logging/setLevel`.

**Example Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "logging/setLevel",
  "params": {
    "level": "debug"
  }
}
```

Available levels (RFC 5424): `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.

### 11.2 Receiving Logs
Logs are sent as notifications from the server. If using **Streamable HTTP**, these arrive via the **SSE stream** (GET request). If using **Stdio**, they are printed to `stdout` as JSON-RPC notifications.

**Example Notification:**
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/message",
  "params": {
    "level": "info",
    "logger": "mcp",
    "data": {
      "msg": "tool execution completed",
      "tool_name": "erp.GET-resource-Employee",
      "duration": "145ms"
    }
  }
}
```

### 11.3 Automatic Redaction
For security, ERPBridge automatically redacts sensitive data from all logs sent to clients. This includes:
- **API Keys & Tokens**
- **Passwords**
- **PII (Emails, Phone Numbers)**
- **Authorization Headers**

Redacted fields will appear as `[REDACTED]` in the log payload. Full, unredacted logs are only available on the server console/logs.

---

## 12. Quick Reference Cheat Sheet

### JSON-RPC Message Template

```json
{
  "jsonrpc": "2.0",
  "id": <any unique integer>,
  "method": "<method name>",
  "params": {}
}
```

### Request Sequence (HTTP)

```
1. POST /mcp/  →  initialize          (no session ID needed)
2. POST /mcp/  →  tools/list          (include Mcp-Session-Id)
3. POST /mcp/  →  tools/call          (include Mcp-Session-Id)
4. GET  /mcp/  →  SSE stream          (optional, include Mcp-Session-Id)
```

### Key Headers

| Header | When to use |
|---|---|
| `Content-Type: application/json` | All POST requests |
| `Mcp-Session-Id: <id>` | All requests after `initialize` |
| `Accept: text/event-stream` | SSE / notifications GET request |

### Transport at a Glance

| | Streamable HTTP | Stdio |
|---|---|---|
| Session ID | Required (from `initialize` response header) | Not used |
| Notifications | Via SSE (GET stream) | Not supported |
| Debugging | Easy (curl, Postman, browser devtools) | Harder |
| Best for | Services, web apps | Local scripts, CLIs |
