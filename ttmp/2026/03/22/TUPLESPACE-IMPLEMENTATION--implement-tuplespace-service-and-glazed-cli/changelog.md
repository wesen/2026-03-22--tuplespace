# Changelog

## 2026-03-22

- Initial workspace created


## 2026-03-22

Created a detailed TupleSpace architecture and implementation guide with a concrete Glazed CLI design based on the imported source note

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/import/tuplespace-plan.md — Primary imported architecture source analyzed in the ticket
- /home/manuel/code/wesen/2026-03-22--tuplespace/ttmp/2026/03/22/TUPLESPACE-IMPLEMENTATION--implement-tuplespace-service-and-glazed-cli/design-doc/01-tuplespace-system-analysis-design-and-implementation-guide.md — Primary design deliverable produced for the ticket


## 2026-03-22

Validated the ticket with docmgr doctor and uploaded the document bundle to /ai/2026/03/22/TUPLESPACE-IMPLEMENTATION on reMarkable

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/ttmp/2026/03/22/TUPLESPACE-IMPLEMENTATION--implement-tuplespace-service-and-glazed-cli/reference/01-investigation-diary.md — Diary records validation output


## 2026-03-22

Uploaded a refreshed validated bundle after recording the final diary and changelog state locally

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/ttmp/2026/03/22/TUPLESPACE-IMPLEMENTATION--implement-tuplespace-service-and-glazed-cli/reference/01-investigation-diary.md — Diary now includes the final validated upload details


## 2026-03-22

Implemented the core module, migrations, matcher, validation, and Postgres store with Docker-backed integration tests (commit d0e7f3b)

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/store/tuple_store.go — Core persistence implementation added in the first code milestone
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/store/tuple_store_test.go — Real Docker-backed store integration tests added in the first code milestone


## 2026-03-22

Implemented the notifier, service, HTTP API, server binary, and Glazed CLI with full-tree tests and manual live smoke checks (commit 707113f)

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespacectl/main_test.go — Built-binary CLI smoke test added and stabilized in the second code milestone
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/service/service.go — TupleSpace runtime semantics implemented in the second code milestone


## 2026-03-22

Added a compact CLI DSL for tuple and template inputs with end-to-end validation against the live server path (commit b7eb804)

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespacectl/cmds/common.go — Added tuple/template DSL parsing and input selection helpers
- /home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespacectl/main_test.go — Extended the built-binary CLI smoke test to exercise the DSL path


## 2026-03-22

Added compose-backed local Postgres startup, moved tuplespaced onto Glazed/Cobra logging, and fixed the notifier idle path so the server stays at low CPU when idle (commit a8c7773)

### Related Files

- /home/manuel/code/wesen/2026-03-22--tuplespace/cmd/tuplespaced/main.go — Reworked tuplespaced into a Glazed bare command with Glazed logging initialization
- /home/manuel/code/wesen/2026-03-22--tuplespace/internal/notify/notifier.go — Replaced the polling-style wait loop with an interruptible notification wait that blocks cleanly when idle
- /home/manuel/code/wesen/2026-03-22--tuplespace/docker-compose.yml — Added local Postgres compose startup for real manual runs
