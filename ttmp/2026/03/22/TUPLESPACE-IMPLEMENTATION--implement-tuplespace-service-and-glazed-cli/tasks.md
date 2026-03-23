# Tasks

## TODO

- [x] Read the imported tuple space design and extract the architecture assumptions that already exist.
- [x] Create a `docmgr` ticket with a primary design doc and an investigation diary.
- [x] Write a detailed intern-oriented analysis, design, and implementation guide including the Glazed CLI plan.
- [x] Bootstrap the Go module, dependency graph, project layout, and local developer entrypoints.
- [x] Add configuration loading, HTTP server startup, graceful shutdown, and migration execution.
- [x] Implement `internal/types` for tuples, templates, bindings, and operation responses.
- [x] Implement validation for spaces, tuples, templates, field kinds, and supported value types.
- [x] Implement Linda-style matcher semantics including repeated formal-name binding checks.
- [x] Implement the Postgres schema and migration for `tuples` and `tuple_fields`.
- [x] Implement tuple insertion, candidate query generation, row decoding, and destructive locking in the store layer.
- [x] Add real Postgres integration tests for the store using Docker-backed test containers.
- [x] Implement the notification fanout with `LISTEN/NOTIFY`, channel normalization, subscription lifecycle, and wakeup tests.
- [x] Implement service-layer `out`, `rd`, `in`, `rdp`, and `inp` orchestration with timeout and cancellation behavior.
- [x] Add service-level concurrency tests proving exactly-once `in` and non-destructive `rd`.
- [x] Implement the HTTP transport, request/response envelopes, error mapping, and health endpoint.
- [x] Add end-to-end HTTP tests against the real Postgres-backed service.
- [x] Implement the `tuplespacectl` Glazed CLI with `tuple out`, `tuple rd`, `tuple in`, and `admin health`.
- [x] Add CLI-level tests and manual smoke checks against a live local service.
- [x] Allow `tuple rd` and `tuple in` to accept multiple positional template specs with one result row per query.
- [x] Update the ticket docs, changelog, and diary with implementation evidence and commit hashes.
- [ ] Close the ticket after implementation work is completed and reviewed.
