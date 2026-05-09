# ERPBridge V2 Architecture: Declarative Control Plane

ERPBridge V2 adopts a **Declarative Control Plane** architecture, inspired by Kubernetes. This design moves away from static, file-system-bound configurations toward a live API-managed resource system for MCP tools.

## 🏗 High-Level Overview

The system is divided into three distinct layers:

1.  **Management Layer (The CLI)**: Developers use `bridgectl` to declare the desired state of the system by "applying" YAML/JSON resource definitions.
2.  **Control Plane (The Server)**: A centralized API server stores tool definitions in a persistent SQLite database, validates them against strict admission rules, and manages versioning.
3.  **Runtime Layer (MCP Engine)**: A background reconciliation controller ensures that the active MCP server always reflects the desired state stored in the database.

---

## 🧩 Endpoint vs. Tool: Key Concepts

One of the most important concepts in ERPBridge V2 is the distinction between registering an **API Endpoint** and applying an **MCP Tool**.

| Feature | **API Endpoint** (`api register`) | **MCP Tool** (`tool apply`) |
| :--- | :--- | :--- |
| **Mental Model** | Technical Discovery | Declarative Management |
| **Primary Focus** | The ERP System (technical) | The AI Agent (semantic) |
| **Storage** | Local CLI Config (`config.yaml`) | Server Registry (`erpbridge.db`) |
| **Visibility** | **Hidden from AI** | **Visible to AI** (as an MCP Tool) |
| **Stability** | Experimental / Internal | Versioned / Stable |
| **Command** | `bridgectl api register ...` | `bridgectl tool apply -f ...` |

### Why separate them?
- **Technical vs. Semantic**: A single ERP API endpoint (e.g., `/api/v1/resource/Employee`) might be used by multiple MCP tools with different filters or versions.
- **Safety**: Registering an API is a developer-only technical step. Applying a tool is a conscious decision to expose functionality to an AI agent.
- **Lifecycle**: You can "discover" and test 100 API endpoints locally, but only "apply" the 5 that are safe and ready for the LLM to use.

---

## 🛠 Core Components

### 1. Tool Resource Registry (The Source of Truth)
Instead of loading files from a directory, the server maintains an internal **Tool Registry** backed by **SQLite**. This registry stores multiple versions of the same tool, allowing for safe rollouts and rollbacks.

### 2. Version Resolver
When an AI agent requests a tool (e.g., `list_employees`), the **Version Resolver** automatically selects the **latest stable version** (e.g., `list_employees@1.2.0`). This ensures that LLMs always interact with consistent, tested schemas even as development continues on newer versions.

### 3. Reconciliation Controller
A background loop that runs inside the ERPBridge server. Every few seconds, it compares:
- **Desired State**: Tools saved in the SQLite database.
- **Actual State**: Tools currently registered in the live MCP server memory.
If a discrepancy is found (e.g., a new tool was applied via CLI), the controller automatically registers/updates the tool in the runtime without requiring a restart.

### 4. Execution Mapping Layer
This layer translates LLM-friendly arguments into ERP-specific technical requirements:
- **Parameter Mapping**: Translates LLM argument names to ERP field names.
- **Credential Resolution**: Resolves `credentialRef` (e.g., `ERP_PRIMARY_KEY`) against environment variables or a vault to inject secrets securely into the outbound HTTP request.
- **Response Unwrapping**: Uses the `responsePath` to extract relevant data from complex ERP payloads before returning them to the LLM.

---

## 🔄 Lifecycle of a Tool Change

1.  **Define**: Developer creates a V2 YAML schema for a new tool.
2.  **Validate**: Runs `bridgectl tool validate -f tool.yaml` to check for syntax and admission rules (e.g., no raw secrets).
3.  **Apply**: Runs `bridgectl tool apply -f tool.yaml`.
4.  **Store**: The ERPBridge API validates the payload again and saves it to the SQLite `tools` table.
5.  **Reconcile**: The background controller detects the new DB entry and registers it with the `mcp-go` runtime.
6.  **Execute**: AI agents now see the new tool and can invoke it immediately.

---

## 🛡 Security Design

- **Secret Decoupling**: Schemas never contain raw tokens or keys. They only contain a reference (`credentialRef`). The middleware resolves these references at the moment of execution using secure environment variables.
- **Admission Controllers**: The API server rejects any tool definition that contains suspicious strings (like `token ` or `key=`) in its endpoint path.
- **Redaction**: All logs produced by tool executions are automatically filtered to redact sensitive keys defined in `internal/types/sensitive.go`.
