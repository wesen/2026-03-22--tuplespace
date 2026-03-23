# Changelog

## 2026-03-22

- Initial workspace created


## 2026-03-22

Created the admin-commands ticket, wrote the implementation plan, and expanded the task list before code changes began

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/ttmp/2026/03/22/TUPLESPACE-ADMIN-COMMANDS--implement-tuplespace-admin-inspection-and-control-commands/design-doc/01-tuplespace-admin-commands-implementation-plan.md — Contains the detailed admin command design and rollout order
- /home/manuel/code/wesen/2026-03-22--tuplespace/ttmp/2026/03/22/TUPLESPACE-ADMIN-COMMANDS--implement-tuplespace-admin-inspection-and-control-commands/reference/01-diary.md — Captures the step-by-step implementation record


## 2026-03-22

Added the shared admin backend plus the first end-to-end read-only admin CLI commands (`spaces`, `dump`, `stats`, `config`, `schema`, `waiters`) with real server-path tests (commit ae1f1c6)

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/admin/models.go — Shared admin request/response data models added for store, service, HTTP, client, and CLI reuse
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/service/admin.go — Added admin service methods, runtime config/schema snapshots, and notifier-backed diagnostics
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/store/admin_store.go — Added tuple listing, counting, lookup, and purge-capable store queries
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/api/httpapi/router.go — Added `/v1/admin/...` route handling
- /home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespacectl/cmds/admin/dump.go — Added one of the new user-facing read-only admin commands
