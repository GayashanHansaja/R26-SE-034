# ERPBridge Documentation

Welcome to the ERPBridge documentation. This wiki-style guide will help you understand, deploy, and extend the middleware.

## 📖 Core Documentation

- **[Architecture Overview](./architecture.md)**: Understanding the Declarative Control Plane, SQLite registry, and reconciliation loop.
- **[Tool Schema Reference (V2)](./tool-schema.md)**: Detailed guide to creating versioned, intent-based MCP tool definitions.
- **[Docker Deployment Guide](./docker.md)**: Detailed instructions for running the full stack using Docker Compose.
- **[Connectivity & Transport Guide](./connectivity.md)**: Understanding SSE, Streamable HTTP, and Direct API transports.
- **[AI Agent Integration](../AGENTS.md)**: Specific patterns for connecting Claude, Cursor, and other agents.

## 🛠 Developer Resources

- **[CLI Reference (bridgectl)](./cli/bridgectl.md)**: Comprehensive guide to the developer CLI.
- **[CLI API Management](./cli/bridgectl_api.md)**: How to register and test ERP endpoints.
- **[CLI Tool Management](./cli/bridgectl_tool.md)**: Managing the live tool registry using `apply`, `get`, and `validate`.
- **[CLI Cache Management](./cli/bridgectl_cache.md)**: Monitoring and flushing the exact match cache.

## 🔌 Integration Guides

- **[Postman Integration](./connectivity.md#postman-configuration)**: Testing MCP endpoints with Postman.
- **[Mock ERP Setup](../mock-erp/README.md)**: Details about the simulated legacy ERP service.
- **[MCP Client Implementation Guide](../docs/mcp-client-guide.md)**: Detailed Guide to Implement MCP Client Use with This MCP Server

## 🛡 System Features

- **Resilience**: Circuit breakers and retry logic (see [README](../README.md#3-resilience--reliability)).
- **[Exact Match Caching](./caching.md)**: High-speed Redis-based storage for ERP responses.
- **Secure Logging**: Real-time log streaming with automatic PII/Secret redaction and RFC 5424 level control.
- **Rate Limiting**: Per-session request throttling for infrastructure protection.
- **Declarative Management**: Versioned tool registry with background reconciliation (no restarts needed).
