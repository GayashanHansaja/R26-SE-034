# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0-alpha.4] (Kotiya) - 2026-05-08

### Added
- feat(mcp): implement custom notification system
- feat(core): add middleware infrastructure and native tool support
- docs: implement wiki-style documentation and docker guide

### Fixed
- fix(mcp): support recursive directory watching for schema hot reloading

### Changed
- refactor(mcp): migrate server logic to middleware-based architecture
- feat(mcp): remove deprecated SSE support in favor of Stdio and Streamable HTTP
- docs: clarify that schema hot reloading supports nested directories

### Improved
- test(mcp): improve test coverage for notifications and endpoints

## [v0.1.0-alpha.3] - 2026-05-08

### Added
- feat(mock-erp): enhance OpenAPI spec for better MCP tool integration
- feat(server): support STDIO transport via --stdio flag and MCP_TRANSPORT env var
- feat(mcp): add client logging, tool notifications, and completion providers
- feat(logger): support redirecting logs to stderr via LOG_TO_STDERR env var
- feat(mcp): implement native streamable http transport support
- docs: add connectivity and transport guide

### Fixed
- fix(docker): resolve build issues and dynamic URL routing
- fix(mcp): resolve tool marshaling conflict by clearing structured schema fields
- fix(logger): use t.Setenv and remove unused os import in tests

### Changed
- refactor(mcp): remove unused resource and prompt completion handlers
- style(cli): standardize error messages and improve context usage
- style(idp): update GenerateFromOpenAPI to accept context
- style(logger): unexport sensitiveKeys to improve encapsulation

### Improved
- docs(cli/mcp/connector): add documentation comments and enable linting
- test: add unit tests for logger, metrics, and cli errors

## [v0.1.0-alpha.2] - 2026-05-07

### Fixed
- fix: include erpbridge-server source and fix gitignore patterns
- fix(goreleaser): update repo owner to nmdra

### Changed
- chore: rename middleware to erpbridge-server
- docs: update README with package details and rename middleware to server

## [v0.1.0-alpha.1] - 2026-05-07

### Added
- feat: add lefthook configuration for automated linting, formatting, and testing
- feat(cli): implement agent-friendly improvements with structured errors and isolated JSON output
- feat: implement schema hot-reloading and bridgectl tool validate
- feat: instrument MCP server with metrics and implement stats endpoint
- feat: implement Prometheus metrics and cache statistics
- feat: add support for MCP Resources and Prompts
- feat: implement circuit breaking and intelligent retries in ERPConnector
- feat: switch embedder to nomic-ai/nomic-embed-text-v1
- feat: implement structured logging with slog, request tracing, and CLI log tailing
- feat: configure cache for finance tools
- feat: add bridgectl cache commands and output formatting
- feat: integrate semantic cache with MCP server and tool execution (internal)
- feat: implement role-aware semantic cache manager
- feat: implement openapi generation and response validation
- feat(cli): implement bridgectl CLI with API and tool management
- feat(middleware): implement MCP bridge middleware and ERP connector
- feat(mock-erp): implement mock ERP server with finance, HR, and inventory modules
- docs: enhance CLI self-documentation and add bridgectl doc generator
- docs: update README with resilience, observability, and DX features
- docs: add .env.example with customizable variables
- docs: add comprehensive guide to README.md
- ci: add GoReleaser release pipeline for middleware and bridgectl

### Fixed
- fix(workflow): downgrade go to 1.24 for golangci-lint compatibility
- fix(cli): resolve numerous lint issues (unchecked errors, body close, etc.)
- fix(idp/connector/config/logger): resolve various lint and stability issues

### Changed
- refactor(monorepo): restructure Go code into services/tools
- chore: prepare for versioning and release
- chore: rename middleware to erpbridge-server
