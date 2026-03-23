---
Title: TupleSpace admin commands implementation plan
Ticket: TUPLESPACE-ADMIN-COMMANDS
Status: active
Topics:
    - tuplespace
    - backend
    - cli
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/tuplespacectl/cmds/admin/dump.go
      Note: Represents the first user-facing read-only admin command built from the plan
    - Path: cmd/tuplespacectl/cmds/admin/root.go
      Note: CLI admin group will host the new commands
    - Path: internal/admin/models.go
      Note: Shared admin contracts introduced in the first implementation slice
    - Path: internal/api/httpapi/router.go
      Note: Admin routes will be added under /v1/admin
    - Path: internal/service/admin.go
      Note: Implements the planned admin service contract and runtime snapshots
    - Path: internal/service/service.go
      Note: Admin design depends on service-owned runtime state and tuple orchestration
    - Path: internal/store/admin_store.go
      Note: Implements the planned admin store queries and destructive filter operations
    - Path: internal/store/tuple_store.go
      Note: Admin plan adds store queries for tuple listing
ExternalSources: []
Summary: ""
LastUpdated: 2026-03-22T21:46:19.189552802-04:00
WhatFor: ""
WhenToUse: ""
---



# TupleSpace admin commands implementation plan

## Executive Summary

This ticket adds a full administrative surface for the TupleSpace server and CLI. The goal is to make the system inspectable and operable without ad hoc SQL, while keeping the core Linda semantics intact. The implementation should expose read-only introspection first, then tuple-targeted maintenance operations, and finally broader control/mutation commands with explicit safeguards.

The server currently exposes only `healthz` plus the tuple-space operations `out`, `rd`, and `in`. The store already contains enough state to implement several inspection commands directly from Postgres, but commands such as `waiters`, `notify-test`, and `stats` require new runtime instrumentation in the service and notifier layers. The design below keeps these concerns separate so the implementation can proceed in testable slices.

## Problem Statement

Operators currently have no first-class way to answer basic questions about a running TupleSpace instance:

- What spaces exist?
- How many tuples are stored?
- What tuples are in a space?
- What is the current runtime configuration?
- What migrations/schema are active?
- Are blocked readers waiting?
- Can I inspect or remove a specific tuple without direct SQL?

Today the only options are:

- manual SQL against the database,
- ad hoc logging,
- indirect inference through client behavior.

That is workable during initial development but weak for debugging, demos, incident handling, and learning the system. The missing admin surface also makes the Glazed CLI feel incomplete because it already has a natural `admin` command group with only one command in it.

## Proposed Solution

Implement a new admin API family under `/v1/admin/...` and corresponding `tuplespacectl admin ...` commands. The new functionality is grouped into four categories.

### 1. Read-only inspection

- `admin spaces`
  - List known spaces with tuple counts and oldest/newest tuple timestamps.
- `admin dump`
  - List tuples for one space or all spaces.
  - Include tuple id, space, arity, created_at, and decoded tuple payload.
- `admin stats`
  - Return service/runtime counters and gauges.
  - Include tuple counts, space counts, current waiters, notifier subscriber counts, candidate limit, and uptime.
- `admin config`
  - Return the effective runtime configuration with database credentials redacted.
- `admin schema`
  - Return migration file names known to the binary plus a lightweight database schema status.

### 2. Tuple-targeted maintenance

- `admin tuple get --tuple-id <id>`
  - Fetch one tuple by internal id.
- `admin tuple delete --tuple-id <id>`
  - Delete one tuple by internal id.

### 3. Filtered inspection/export

- `admin peek`
  - Non-destructive admin listing with filters:
    - `--space`
    - `--limit`
    - `--offset`
    - `--created-before`
    - `--created-after`
- `admin export`
  - Same backing query as `peek`, but optimized for bulk output and intended for JSON/NDJSON emission.

### 4. Control and runtime diagnostics

- `admin purge`
  - Delete tuples matching admin filters.
  - Must require an explicit safety flag such as `--confirm`.
- `admin waiters`
  - Show currently blocked `rd`/`in` operations tracked by the service.
- `admin notify-test`
  - Trigger a test notification on a space channel and report notifier state.

### API shape

The existing tuple endpoints use the `/v1/spaces/{space}/...` namespace. The admin surface should not overload that path because many admin commands are cross-space or server-scoped. The proposed routes are:

- `GET /v1/admin/spaces`
- `POST /v1/admin/dump`
- `GET /v1/admin/stats`
- `GET /v1/admin/config`
- `GET /v1/admin/schema`
- `GET /v1/admin/tuples/{id}`
- `DELETE /v1/admin/tuples/{id}`
- `POST /v1/admin/peek`
- `POST /v1/admin/export`
- `POST /v1/admin/purge`
- `GET /v1/admin/waiters`
- `POST /v1/admin/notify-test`

Use `GET` for read-only, parameter-light endpoints and `POST` where filters are easier to express in JSON bodies.

### Shared data structures

New internal read models should include:

- `AdminTupleRecord`
  - `id`
  - `space`
  - `arity`
  - `created_at`
  - `tuple`
- `SpaceSummary`
  - `space`
  - `tuple_count`
  - `oldest_tuple_at`
  - `newest_tuple_at`
- `WaiterInfo`
  - `id`
  - `space`
  - `operation`
  - `wait_ms`
  - `started_at`
  - `template`
- `StatsSnapshot`
  - `started_at`
  - `uptime_ms`
  - `space_count`
  - `tuple_count`
  - `waiter_count`
  - `notifier_channels`
  - `notifier_subscribers`
  - `candidate_limit`

### Store layer additions

The Postgres store should gain:

- `ListSpaces(ctx, q)`
- `ListTuples(ctx, q, filter)`
- `GetTupleByID(ctx, q, id)`
- `DeleteTupleByID(ctx, tx, id)`
- `DeleteTuples(ctx, tx, filter) -> count`
- `CountTuples(ctx, q, filter)`

Use one filter struct shared by `dump`, `peek`, `export`, and `purge`:

- optional `space`
- optional `created_before`
- optional `created_after`
- optional `limit`
- optional `offset`

### Service-layer additions

The `service.TupleSpace` interface should expand into a richer service contract for admin operations. The concrete `Service` already has access to the database pool, store, notifier, and candidate limit, so it is the correct home for:

- query orchestration,
- waiter instrumentation,
- config snapshots,
- uptime tracking,
- schema/migration reporting.

### Waiter instrumentation

`waiters` is the one command that cannot be implemented from the database alone. The service currently blocks inside `read(...)` when `rd` or `in` use a positive wait. Add a waiter registry to `Service`:

- assign a waiter id when entering a blocking wait,
- record:
  - operation kind,
  - space,
  - normalized template,
  - wait duration,
  - start time,
- remove the waiter on success, timeout, cancellation, or shutdown.

The registry can be an in-memory mutex-protected map because it is purely runtime state.

### Notifier introspection

The notifier currently tracks refcounts and subscribers internally. Add a snapshot method that returns:

- channel count,
- total subscriber count,
- per-channel subscriber counts.

Also add a safe `Notify(space string)` helper so `notify-test` can exercise the same channel naming path without duplicating SQL in multiple layers.

## Design Decisions

### Admin endpoints live under `/v1/admin/...`

Rationale:

- separates operator APIs from normal tuple semantics,
- avoids mixing cross-space commands into `/v1/spaces/{space}/...`,
- keeps future auth policy decisions straightforward.

### `dump`, `peek`, and `export` share one underlying filtered listing path

Rationale:

- prevents three near-duplicate store queries,
- keeps CLI behavior consistent,
- makes test coverage easier because the query semantics are centralized.

### `waiters` is runtime-only state, not persisted state

Rationale:

- blocked readers are transient and process-local,
- persisting them would add failure modes and cleanup complexity,
- the immediate goal is operational debugging, not distributed coordination.

### `purge` requires explicit confirmation

Rationale:

- broad destructive commands should not be one typo away,
- a human-visible safety rail is more important than perfect automation here.

### `schema` reports binary-known migrations plus database shape checks, not a full migration ledger

Rationale:

- the current migration system is file-application based and does not record applied versions,
- introducing a real migration table would be a bigger architectural change than this ticket needs,
- operator value can still be delivered by listing packaged migration files and checking that required tables/indexes exist.

### `notify-test` stays narrow

Rationale:

- the goal is diagnosing notification plumbing, not building a generic event bus,
- a simple “send test wakeup for this space” command is enough.

## Alternatives Considered

### Put all admin commands directly in the CLI with SQL calls

Rejected because:

- it bypasses the server contract,
- it would require database credentials on every CLI user,
- it splits operational logic across the client and server.

### Add only `dump` and `spaces`, leave everything else for later

Rejected because the user explicitly asked for the full proposal set, and several commands share the same underlying scaffolding. Building the broader service contract now is more coherent than serial ad hoc additions.

### Persist waiter state in Postgres

Rejected because:

- waiter state is transient,
- it would complicate cleanup on crashes,
- it does not improve the core operator use case enough to justify the complexity.

### Implement `export` as a CLI-only alias for `dump`

Rejected because:

- export semantics usually grow separately from human inspection,
- keeping a dedicated endpoint avoids overfitting `dump` to both operator and tooling use cases.

## Implementation Plan

### Phase 1: shared models and read-only foundations

- Add admin request/response types under `internal/api/httpapi`.
- Add store query/filter structs and read-model structs.
- Implement store methods for:
  - space summaries,
  - tuple listing,
  - tuple lookup by id,
  - tuple counts.
- Add unit/integration tests for those queries.

### Phase 2: service instrumentation and admin read APIs

- Add service start time and config snapshot storage.
- Add waiter registry and notifier snapshot support.
- Implement service methods for:
  - spaces,
  - dump,
  - stats,
  - config,
  - schema,
  - waiters.
- Add service tests for waiter lifecycle and stats snapshots.

### Phase 3: HTTP handlers and read-only CLI commands

- Extend the router with `/v1/admin/...`.
- Add handlers for:
  - spaces,
  - dump,
  - stats,
  - config,
  - schema,
  - waiters.
- Extend the client package with matching helpers.
- Implement CLI commands:
  - `admin spaces`
  - `admin dump`
  - `admin stats`
  - `admin config`
  - `admin schema`
  - `admin waiters`

### Phase 4: tuple-targeted and filtered tuple operations

- Implement service/store/HTTP/client/CLI support for:
  - `tuple get`
  - `tuple delete`
  - `peek`
  - `export`
- Add destructive tests proving delete-by-id works and does not affect other tuples.

### Phase 5: purge and notify diagnostics

- Implement `purge` filter handling and confirmation requirements.
- Implement `notify-test`.
- Add tests for:
  - confirm gating,
  - filtered purge counts,
  - notifier wakeup path.

### Phase 6: full validation and documentation

- Add built-binary CLI tests for the new command set.
- Run `go test ./... -count=1`.
- Update diary, changelog, and task checklist after each slice.
- Run `docmgr doctor --ticket TUPLESPACE-ADMIN-COMMANDS --stale-after 30`.

## Open Questions

- Should `dump` default to all spaces or require `--space` unless `--all-spaces` is present?
  - Proposed answer: allow all spaces, but sort by `space, id`.
- Should `export` support writing directly to a file in the first implementation?
  - Proposed answer: no dedicated file flag yet; Glazed output redirection is enough.
- Should `purge` support tuple-template matching in v1?
  - Proposed answer: no. Start with space/time filters and add template filters later if needed.
- Should `schema` expose index names exactly or summarize them?
  - Proposed answer: include exact names so the output is copy/paste-friendly for debugging.

## References

- Current CLI admin root: `/home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespacectl/cmds/admin/root.go`
- Current HTTP router: `/home/manuel/code/wesen/2026-03-22--tuplespace/internal/api/httpapi/router.go`
- Current service core: `/home/manuel/code/wesen/2026-03-22--tuplespace/internal/service/service.go`
- Current store core: `/home/manuel/code/wesen/2026-03-22--tuplespace/internal/store/tuple_store.go`
- Current schema: `/home/manuel/code/wesen/2026-03-22--tuplespace/migrations/001_init_tuplespace.sql`
