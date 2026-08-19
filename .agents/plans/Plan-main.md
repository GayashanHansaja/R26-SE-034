# Plan: Hardening — Cache, Security, Correctness, CLI & Docs

## Goal

Remediate the non-auth findings from the repository analysis: caching works without Redis, cache keys cannot collide, the identified security holes close, correctness defects fix, and CLI/docs artifacts stop drifting from reality. Auth/tokens are **out of scope** (see `.agents/plans/Plan.md`).

## Current State (evidence)

**Cache**
- No Redis ⇒ no cache at all: `cacheMgr` stays nil when `REDIS_URL` empty (services/erpbridge-server/main.go:55,77-90); `CacheMiddleware` bypasses on `s.cache == nil` (internal/mcp/middleware.go:137). Docs say "cache disabled" (docs/caching.md:94, docs/faq.md:40).
- Keys truncated to 32 bits: `argsHash` uses `h[:4]` → 8 hex chars (internal/cache/manager.go:96-113).
- `CachedAt` fabricated at read time; `EnsureIndex` is a no-op (manager.go:43-46); `scanAndDelete` DELs one key per round-trip (internal/cache/flush.go:76-83); `handleCacheList`/`handleCacheInspect` are 501 stubs (internal/mcp/server.go:812-818).

**Security**
- Silent credential fallback: `os.Getenv(CredentialRef)` falls back to the literal ref string as the credential (internal/mcp/tool.go:199-203).
- Redaction only on the MCP stream handler (internal/logger/mcp_handler.go); `broadcastHandler` (internal/logger/logger.go:52-87) and `LoggingMiddleware` (internal/mcp/middleware.go:91 — logs full `req.Params.Arguments`) emit raw values.
- Hardcoded credential in docker-compose.yml:44 (`ERP_PRIMARY_KEY=token adm_key_001:adm_sec_stu901`).

**Correctness**
- Ghost tools: deactivated tools hidden only by the HTTP SSE line filter (internal/mcp/server.go:370, 477-531); Stdio transport advertises them (main.go:111 uses `mcp_server.ServeStdio`).
- Weak state hash: count+activeSum+maxUpdated (internal/mcp/store.go:121).
- Migration ignores errors: `_, _ = s.db.Exec("ALTER TABLE tools ADD COLUMN is_active ...")` (store.go:60).
- Unbounded per-session rate limiter map (internal/mcp/middleware.go:35-51).
- ResponsePath: top-level map key only; missing path silently returns full response (internal/mcp/tool.go:226-233).

**CLI & docs**
- `log tail` filters via `strings.Contains` on raw JSON (internal/cli/log.go:146-160).
- `tool get`/`describe` fetch the whole tool list and filter client-side (internal/cli/tool.go:106-235); server `handleToolList` has no query filters (internal/mcp/server.go:620).
- idp registry is a plain JSON file, no locking (internal/idp/registry.go:39, 62-66).
- Parse errors silently ignored in `tool validate` (internal/cli/tool.go:286-291).
- Drift: Postman + mcp-client-guide use `finance.list_invoices_api_v1_finance_invoices_get`; skill `mcp-tool.yaml` uses `spec.endpoint.ref/method` (actual: `spec.execution.method/endpoint`); docs list stale `SCHEMAS_DIR`/`EMBEDDER_URL`; generated `outputSchema` carries unresolvable `$ref` to OpenAPI components (internal/idp/generator.go:258-267).

## Decisions

| # | Decision | Rationale / rejected |
|---|---|---|
| B-D1 | Cache backend interface: `Backend` (Get/Set/FlushTool/FlushModule/AutoFlush/Stats) with `RedisBackend` (existing go-redis logic) and `MemoryBackend` (mutex LRU, default 10,000 entries, `CACHE_MEMORY_MAX_ENTRIES` env, honors per-tool `Config.TTLSeconds`). Auto-selection: `REDIS_URL` set ⇒ Redis, else Memory. **No `CACHE_BACKEND` override env** (minimal surface). | Single-struct branching rejected (muddles concerns); explicit backend override rejected (unneeded config). |
| B-D2 | Full SHA-256 hex (64 chars) cache keys (manager.go:112). Old short keys become harmless misses; no migration. | 128-bit still theoretical collisions; full hash is free. |
| B-D3 | `CachedAt` stored via JSON envelope `{response, cachedAt}`; decode failure ⇒ miss (handles old raw-byte entries). | Dropping the field breaks the Entry contract. |
| B-D4 | Remove `EnsureIndex` + call site (main.go:85) + tests; remove 501 stub handlers + their routes (server.go:812-818, 562-563); batch `UNLINK` (100/iter) in `scanAndDelete`. | Dead code; unused surface; flush speed. |
| B-D5 | credentialRef fail-closed: `CredentialRef` set + env missing ⇒ error `credentialRef X not found in environment`; literal fallback only when ref empty; startup warning lists affected schemas. | Stops secret leakage through the registry. |
| B-D6 | Shared redaction: extract masq `ReplaceAttr` from mcp_handler.go into `logger.RedactAttr`; apply in `broadcastHandler`; add `logger.RedactArgs(map)` (sensitive keys: password/token/api_key/secret/key/ssn/authorization…) used by `LoggingMiddleware`. | Console/broadcast logs stay leaky otherwise. |
| B-D7 | Compose credential → `${ERP_PRIMARY_KEY:-}` interpolation (docker-compose.yml:44); dev value only in `.env.example`. | No secrets in tracked config. |
| B-D8 | Ghost tools: extract `filterToolsList` (server.go:514-531) into one helper; HTTP path uses it (as today); Stdio path wraps stdout via `server.NewStdioServer(server.MCPServer()).Listen(ctx, os.Stdin, filterWriter)` (mcp-go v0.57.0 has no `RemoveTool` — verified vendor/server.go:187-236; `Listen(ctx, reader, writer)` is public, vendor/.../stdio.go:504). | Only viable mechanism without forking mcp-go; keeps both transports consistent. |
| B-D9 | State hash: SHA-256 over sorted `(name, version, is_active, updated_at)` tuples from `SELECT name, version, is_active, updated_at FROM tools ORDER BY name, version`. | Renames/timestamp ties no longer collide. |
| B-D10 | Migration: `PRAGMA table_info(tools)` check before `ALTER TABLE ADD COLUMN is_active`; add only if missing; propagate real errors (store.go:40-64). | Hides real failures today. |
| B-D11 | Rate limiter: track lastSeen; when map > 10k, sweep entries idle > 15 min (lazy, inside `getLimiter`, middleware.go:35-51). | No background goroutine. |
| B-D12 | ResponsePath: dotted paths + `[i]` array indexes; missing path ⇒ tool-call error (behavior change from silent full-response, tool.go:226-233). | Deterministic; schema drift becomes visible. |
| B-D13 | CLI: parse SSE lines as JSON and filter on parsed fields (log.go:146); `handleToolList` accepts `?name=`/`?version=` and CLI passes them; idp registry atomic write (temp + rename) + in-process mutex; `tool validate` surfaces parse errors. | Structured filtering; server-side filtering; crash-safe registry; no silent failures. |
| B-D14 | Docs/artifacts: fix Postman + mcp-client-guide tool names; rewrite skill `mcp-tool.yaml` to `spec.execution.*`; align skill flags with onboarding; prune `SCHEMAS_DIR`/`EMBEDDER_URL`; document `MOCK_ERP_LOG_LEVEL`/`BRIDGE_AUTH_*`; generator emits dereferenced `outputSchema` (no `$ref`). | New users follow correct examples. |

## Scope

**In:** cache (backend interface, memory fallback, full keys, envelope, hygiene), security (credentials, redaction, compose), correctness (ghost tools, state hash, migration, limiter, ResponsePath), CLI + docs drift.

**Out:** auth/tokens (Plan.md), OAuth, mock ERP improvements (tests/latency/persistence), cache `list`/`inspect` implementation, `CACHE_BACKEND` override, per-token rate limits, notification renaming.

## Tasks

- [ ] **C1 — Cache backend interface + memory fallback** (**Seam:** all `cache.Manager` methods consumed by middleware.go:134-176 and server.go:791-810; **Files:** internal/cache/manager.go (interface + Config), internal/cache/redis_backend.go, internal/cache/memory_backend.go (LRU: sync.Mutex + map + container/list or simple map+lastUsed; cap 10k default, `CACHE_MEMORY_MAX_ENTRIES`; per-key TTL from `Config.TTLSeconds`; prefix-scan flush), internal/mcp/middleware.go (type of `s.cache`), internal/mcp/server.go (field type), services/erpbridge-server/main.go:77-90 (selection), internal/cache/manager_test.go, memory_backend_test.go; **Verify:** `go test ./internal/cache/... ./internal/mcp/...`; manual: run without `REDIS_URL`, call a cached tool twice, second call logs `cache hit` + increments `metrics.CacheHitsTotal`; `/api/cache/stats` returns entry count instead of 503).
- [ ] **C2 — Full-hash keys + CachedAt envelope** (**Seam:** `argsHash` (manager.go:96) and `Set`/`exactGet`; **Files:** internal/cache/manager.go, internal/cache/exact.go, manager_test.go (update TestArgsHash; add distinct-args-never-collide test), exact_test.go; **Verify:** `go test ./internal/cache/...`; old raw-byte entries decode as miss without error).
- [ ] **C3 — Cache hygiene** (**Files:** internal/cache/manager.go (drop `EnsureIndex`), services/erpbridge-server/main.go:85 (drop call), internal/cache/flush.go:76-83 (batch `UNLINK` via pipeline, 100/iter), internal/mcp/server.go:812-818 (delete stubs) + :562-563 (delete routes), cache flush tests; **Verify:** `go vet ./...`; `rg -n "handleCacheList|handleCacheInspect" internal/` clean).
- [ ] **S1 — credentialRef fail-closed** (**Seam:** `Tool.Execute`, internal/mcp/tool.go:198-203; **Files:** internal/mcp/tool.go, tool_test.go (add: non-empty ref + missing env ⇒ error; empty ref ⇒ no auth header), services/erpbridge-server/main.go (startup warning — enumerate store tools whose `CredentialRef` is non-empty and env-unset); **Verify:** `go test ./internal/mcp/ -run TestTool_Execute`; affected tool fails with clear error).
- [ ] **S2 — Shared log redaction** (**Seam:** `logger.Init` multi-handler wiring (logger.go:106-118) + `LoggingMiddleware`; **Files:** internal/logger/mcp_handler.go (extract redaction to `RedactAttr`), internal/logger/logger.go:52-87 (use it in broadcastHandler), internal/logger/redact.go (new: `RedactArgs(any) any` recursive map redactor), internal/mcp/middleware.go:91 (redact args), internal/logger/logger_test.go, internal/mcp/middleware_test.go; **Verify:** `go test ./internal/logger/... ./internal/mcp/...`; broadcast output and tool-args logs contain `[REDACTED]`, never the values; mcp_handler tests still pass).
- [ ] **S3 — Compose credential** (**Files:** docker-compose.yml:44 → `ERP_PRIMARY_KEY=${ERP_PRIMARY_KEY:-}`, .env.example stays as documented dev value; **Verify:** `docker compose config | rg ERP_PRIMARY_KEY` shows the interpolation, no literal secret).
- [ ] **K1 — Ghost tools on Stdio** (**Seam:** `DeregisterTool`/`RegisterTool` (server.go:364-425) + stdio startup (main.go:109-111); **Files:** internal/mcp/server.go (extract `filterToolsList(tools []mcp.Tool) []mcp.Tool` from 514-531), internal/mcp/filter_writer.go (new: `toolsListFilterWriter{out io.Writer, filter func([]mcp.Tool) []mcp.Tool}` — scans newline-delimited JSON-RPC output, rewrites `tools/list` result lines, forwards everything else byte-for-byte), services/erpbridge-server/main.go (replace `mcp_server.ServeStdio(server.MCPServer())` with `server.NewStdioServer(...).Listen(ctx, os.Stdin, filterWriter{os.Stdout})`), server_test.go (both transports omit deactivated tools); **Verify:** `go test ./internal/mcp/ -run TestServer_Reconcile`; manual smoke: delete a tool, run `erpbridge-server --stdio` + an MCP stdio client, `tools/list` omits it; re-apply restores it).
- [ ] **K2 — Robust state hash** (**Seam:** `Store.GetStateHash` (store.go:121); **Files:** internal/mcp/store.go, store_test.go (add: same count/sum but renamed tool ⇒ hash changes); **Verify:** `go test ./internal/mcp/ -run TestStore_GetStateHash`).
- [ ] **K3 — Safe migration** (**Seam:** `Store.init` (store.go:40-64); **Files:** internal/mcp/store.go (PRAGMA check), store_test.go (fixture: create table without `is_active`, assert init adds it and preserves rows); **Verify:** `go test ./internal/mcp/ -run TestStore_NewStore`).
- [ ] **K4 — Limiter eviction** (**Seam:** `RateLimitMiddleware.getLimiter` (middleware.go:35-51); **Files:** internal/mcp/middleware.go (lastSeen map, sweep at >10k entries, 15-min idle), middleware_test.go; **Verify:** `go test ./internal/mcp/ -run TestRateLimitMiddleware`; eviction unit test).
- [ ] **K5 — ResponsePath hardening** (**Seam:** tool.go:226-233; **Files:** internal/mcp/tool.go (resolve dotted paths + `[i]`; error when path missing), tool_test.go (nested, array, missing-path cases); **Verify:** `go test ./internal/mcp/ -run TestTool_Execute`).
- [ ] **D1 — Structured log tail** (**Seam:** `shouldPrint` (cli/log.go:146-160); **Files:** internal/cli/log.go (json.Unmarshal each `data:` message, filter on `component`/`tool_name`/`level`/`request_id` fields; malformed lines pass through), log_test.go; **Verify:** `go test ./internal/cli/ -run TestLogTail`; substring false-positives gone).
- [ ] **D2 — Server-side tool filter** (**Seam:** `handleToolList` (server.go:620); **Files:** internal/mcp/server.go (`?name=` exact + `?version=`; empty = all), internal/cli/tool.go (get/describe pass name@version parsed by `mcp.ParseToolIdentifier`), api_test.go, tool_test.go; **Verify:** `go test ./internal/mcp/ ./internal/cli/`; `get` sends query params).
- [ ] **D3 — Registry atomic writes + validate errors** (**Seam:** `idp.Registry.save` (registry.go:62-66) + `tool validate` (cli tool.go:276-304); **Files:** internal/idp/registry.go (mutex + temp-file + `os.Rename`), internal/idp/registry_test.go (concurrent save, invalid JSON), internal/cli/tool.go (surface unmarshal errors), tool_test.go; **Verify:** `go test ./internal/idp/ ./internal/cli/`; invalid schema file yields an error, not silence).
- [ ] **D4 — Docs/artifact drift + generator deref** (**Files:** erpbridge_postman_collection.json + docs/mcp-client-guide.md (rename `finance.list_invoices_api_v1_finance_invoices_get` → current `erp.list_purchase_invoices` etc.), skills/bridgectl-add-api/assets/mcp-tool.yaml (`spec.execution.method/endpoint`), skills/bridgectl-add-api/SKILL.md (flag parity with docs/onboarding.md), docs/environment-variables.md (drop `SCHEMAS_DIR`/`EMBEDDER_URL`; add `MOCK_ERP_LOG_LEVEL`, `BRIDGE_AUTH_*`, `CACHE_MEMORY_MAX_ENTRIES`), internal/idp/generator.go:258-267 (emit dereferenced `Schema.Value` — resolve `$ref` via kin-openapi before marshaling), generator_test.go (assert no `$ref` in outputSchema), CHANGELOG.md; **Verify:** `rg -n "list_invoices_api|spec\.endpoint|SCHEMAS_DIR|EMBEDDER_URL" --glob '!vendor/**'` clean; `bridgectl tool generate --api erp --openapi mock-erp/openapi.yaml` produces `$ref`-free outputSchema).

## Verification

1. `make test && make lint` green.
2. **No-Redis smoke**: server without `REDIS_URL`; repeat cached tool call ⇒ second call cache hit (log + metric); `/api/cache/stats` live.
3. **Ghost-tool smoke**: delete a tool ⇒ `tools/list` omits it over both Stdio and HTTP; re-apply restores.
4. **Redaction smoke**: tool call with `password`-keyed arg ⇒ `bridgectl log tail` and console show `[REDACTED]`.
5. **Docs audit**: `rg` from D4 clean; regenerated schemas `$ref`-free.
6. **CLI hygiene**: no bare `http.Get`/`DefaultClient` outside shared helpers (audit with `rg`).

## Open Questions

None — decisions B-D1..B-D14 settled during the design-tree (grilling) session. Implementation order: C1 → C2 → C3 (cache), S1 → S2 → S3 (security), K1 → K5 (correctness), D1 → D4 (CLI/docs). Plan A (auth) is a separate plan in `.agents/plans/Plan.md`; run it first (priority feature) unless stated otherwise.

## Review Addendum (2026-08-20) — code-vs-plan audit

All claims verified against the codebase. Findings below.

### Line-number drift

Files have grown since the plan was written. Locate seams by code pattern, not line number:

| File | Plan max reference | Actual line count |
|---|---|---|
| `internal/mcp/server.go` | ~818 | 873 |
| `internal/mcp/store.go` | ~121 | 177 |
| `internal/mcp/middleware.go` | ~176 | 176 (unchanged) |
| `internal/mcp/tool.go` | ~233 | 267 |
| `internal/cache/manager.go` | ~113 | 151 |
| `internal/cache/flush.go` | ~83 | 84 (unchanged) |
| `services/erpbridge-server/main.go` | ~111 | 128 |

Most references are off by 1–10 lines. Search by function name or string literal.

### Verified claims (all confirmed)

- ✅ `argsHash` uses `h[:4]` → 8 hex chars (manager.go:112)
- ✅ `EnsureIndex` is a no-op, still called from main.go:85
- ✅ credentialRef fallback sends env var *name* as credential (tool.go:199-203) — **real vulnerability**
- ✅ `LoggingMiddleware` logs raw `req.Params.Arguments` with no redaction (middleware.go:91)
- ✅ Hardcoded `ERP_PRIMARY_KEY=token adm_key_001:adm_sec_stu901` in docker-compose.yml:44
- ✅ Stdio path uses `mcp_server.ServeStdio()` with no tool filter (main.go:111)
- ✅ State hash is `count-activeSum-maxUpdated` — renamed tools don't change hash (store.go:121-130)
- ✅ Migration error swallowed: `_, _ = s.db.Exec("ALTER TABLE ...")` (store.go:60)
- ✅ Rate limiter map never evicts (middleware.go:35-51)
- ✅ ResponsePath is top-level map key only; missing path silently returns full response (tool.go:226-233)

### Risk assessments

- **K1 (Ghost tools on Stdio) — HIGH RISK**: The `filterWriter` approach parses streaming JSON-RPC output from the MCP SDK's internal wire format. If `mcp-go` changes serialization (buffering, framing), this breaks silently. **Mitigation**: pin `mcp-go` version tightly; add an integration test that verifies the wire format assumption; test partial writes and buffered lines.
- **S1 (credentialRef fail-closed) — HIGHEST PRIORITY**: This is the most impactful security fix in either plan — a typo in `credentialRef` silently sends the env var name as a password. Simple fix, zero dependencies. **Recommend implementing first.**
- **C1 (cache backend interface) — LARGEST TASK**: Consider splitting into two commits: (1) extract the `Backend` interface + `RedisBackend`, (2) add `MemoryBackend`.

### Cancelled tasks — known limitations to document

- **C6 (FlushModule patterns)**: The bug is real — `FlushModule` produces `exact:<module>.*:*` but keys are `exact:<tool>:<role>:<hash>` (no module prefix). `bridgectl cache flush --module X` silently does nothing. Acceptable to defer but should be documented as a known limitation.
- **K6 (Resource relative URLs)**: `resource.go:195` hardcodes `http://localhost:8081`. Acceptable if resources are rarely used.

### Recommended cross-plan ordering

Run security fixes first regardless of plan priority:

```
1. S1 (credentialRef fail-closed)     — highest-impact security fix, zero deps
2. S3 (compose credential)            — one-liner security fix
3. S2 (shared log redaction)          — security, before auth logs more data
4. Plan.md A1–A6                      — auth core
5. Plan.md A7–A8                      — bridgectl token + api-token plumbing
6. C1 → C2 → C3                      — cache (independent of auth)
7. K2 → K3 → K4 → K5                 — correctness (low-risk)
8. K1 (ghost tools on stdio)          — high-risk, do after K2-K5 stable
9. D1 → D2 → D3                      — CLI (build on A8's newRequest helper)
10. Plan.md A9, A10 + D4              — CORS, docs (final pass)
```


## Audit Addendum (2026-08-19) — new findings from `Report-code-vs-docs-audit.md` (erpbridge-docs repo)

Verified against code; each claim checked. These tasks are additive to C1-C3/S1-S3/K1-K5/D1-D4 above.

- [ ] **C4 — Auto-flush must run (CANCELLED per user: no code changes) on write tools with cache disabled** (**Seam:** `CacheMiddleware` bypass, internal/mcp/middleware.go:137-139; **Files:** internal/mcp/middleware.go (bypass only when `cache == nil`, or `Spec.Cache == nil`, or cache disabled AND `len(FlushOn)==0`), middleware_test.go (write tool with `enabled:false` + `flushOn` ⇒ flush called; cacheable tool unchanged); **Verify:** `go test ./internal/mcp/ -run TestCacheMiddleware`).
- [ ] **C5 — Accept `invalidateOn` (CANCELLED per user: no code changes) alias in cache Config** (**Seam:** `cache.Config` (manager.go:18-23) — `invalidateOn` is silently dropped by `json.Unmarshal`; **Files:** internal/cache/manager.go (custom `UnmarshalJSON` merging `flushOn` + `invalidateOn`), manager_test.go (both spellings populate `FlushOn`; unknown keys ignored); **Verify:** `go test ./internal/cache/ -run TestConfig_Unmarshal`).
- [ ] **C6 — Fix FlushAll/FlushModule (CANCELLED per user: no code changes) key patterns** (**Seam:** flush.go:34-41 + server.go:769 — `FlushModule(ctx,"")` produces `exact:.*:*` (`.` literal ⇒ matches nothing) and `FlushModule(module)` produces `exact:<module>.*:*` but keys are `exact:<tool>:<role>:<hash>` with no module prefix (manager.go:92); **Files:** internal/cache/flush.go (add `FlushAll` scanning `exact:*`; keep `FlushModule` only for the index-free path — see decision), internal/mcp/server.go (all=true ⇒ `FlushAll`; module ⇒ resolve module→stable tool names from registry then `FlushTool` per name), flush_test.go (miniredis: flush-all deletes all exact keys; module flush deletes only that module's tools' keys); **Verify:** `go test ./internal/cache/ ./internal/mcp/`; manual: seed 2 tools in different modules, `/api/cache/flush?all=true` ⇒ 0 stale keys).
- [ ] **K6 — Resource relative URLs (CANCELLED per user: no code changes) must honor `ERP_BASE_URL`** (**Seam:** internal/mcp/resource.go:50-52 hardcodes `http://localhost:8081`; tool.go:173-196 resolves env properly; **Files:** internal/mcp/resource.go (relative path ⇒ `os.Getenv("ERP_BASE_URL")` default `http://localhost:8081`), resource_test.go (env set ⇒ joined URL; unset ⇒ default); **Verify:** `go test ./internal/mcp/ -run TestResource`).
- [ ] **D5 — Remove shadowed (CANCELLED per user: no code changes) `-o/--output` on `tool get`** (**Seam:** internal/cli/tool.go:430 re-declares root persistent flag (root.go:105) with different help text; **Files:** internal/cli/tool.go (drop local flag), tool_test.go (help renders one `-o`), regenerate `docs/bridgectl/` cobra pages in erpbridge-docs if the usage line changes; **Verify:** `go test ./internal/cli/`; `bridgectl tool get --help | rg -c -- '-o, --output'` ⇒ 1).
