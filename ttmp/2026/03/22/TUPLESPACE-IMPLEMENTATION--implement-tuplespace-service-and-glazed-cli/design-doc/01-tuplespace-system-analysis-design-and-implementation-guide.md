---
Title: TupleSpace System Analysis, Design, and Implementation Guide
Ticket: TUPLESPACE-IMPLEMENTATION
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../corporate-headquarters/glazed/pkg/cli/cli.go
      Note: Reference for the command settings section exposed to Glazed commands
    - Path: ../../../../../../../corporate-headquarters/glazed/pkg/cli/cobra.go
      Note: Reference for building Cobra commands from Glazed command descriptions
    - Path: ../../../../../../../corporate-headquarters/glazed/pkg/cmds/cmds.go
      Note: Reference for NewCommandDescription and command description structure
    - Path: ../../../../../../../corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md
      Note: Reference for structuring the Glazed CLI command constructors
    - Path: cmd/tuplespacectl/main.go
      Note: Implements the Glazed CLI entrypoint and command tree described in the design guide
    - Path: import/tuplespace-plan.md
      Note: Primary imported design source for TupleSpace semantics
    - Path: internal/api/httpapi/router.go
      Note: Implements the HTTP surface described in the design guide
    - Path: internal/notify/notifier.go
      Note: Implements the LISTEN/NOTIFY fanout design used by blocking reads and consumes
    - Path: internal/service/service.go
      Note: Implements TupleSpace operation semantics and blocking retry loops
    - Path: internal/store/tuple_store.go
      Note: Implements tuple persistence
ExternalSources: []
Summary: Detailed intern-facing design for implementing a Linda-style TupleSpace service in Go with Postgres persistence, LISTEN/NOTIFY wakeups, and a Glazed-based CLI.
LastUpdated: 2026-03-22T12:30:39.971627682-04:00
WhatFor: Detailed implementation guide for turning the imported tuple space proposal into a production-shaped Go service plus a Glazed-based command line client.
WhenToUse: Use when implementing the tuple space service, reviewing architecture, onboarding a new engineer, or wiring the Glazed CLI.
---



# TupleSpace System Analysis, Design, and Implementation Guide

## Executive Summary

The imported source file describes a strong first-pass architecture for a Linda-style tuple space: a Go service exposing `out`, `rd`, and `in`; Postgres as the source of truth; Go-based semantic matching; and `LISTEN/NOTIFY` to wake blocked readers and consumers. That proposal is the right foundation because it keeps concurrency control in one place, uses Postgres row locks for destructive reads, and avoids trying to encode full tuple semantics directly in SQL.

What the imported file does not yet provide is the implementation envelope around that core: transport contracts, request and error shapes, package boundaries beyond a sketch, migration layout, validation rules, observability, test strategy, and the command line utility the user requested. This document fills those gaps and turns the sketch into an implementation-ready plan for an intern who needs both conceptual orientation and file-by-file execution guidance.

The main recommendation is:

1. Build an HTTP-first service binary called `tuplespaced`.
2. Keep Postgres as canonical storage with `tuples` plus `tuple_fields`.
3. Implement semantic matching in Go only after a SQL candidate filter.
4. Use a dedicated notification component backed by one long-lived `LISTEN` connection.
5. Add a separate Glazed-based CLI binary called `tuplespacectl` for `out`, `rd`, `in`, and operational inspection.

This keeps the minimum viable system small while still giving the codebase clean seams for later additions such as gRPC, more types, richer observability, and multi-space operational tooling.

## Reading Guide For A New Intern

Read this document in the following order if you are new to tuple spaces or new to the codebase:

1. Read "Problem Statement and Scope" to understand what the system is meant to do.
2. Read "Current-State Analysis From The Imported Plan" to see what is already decided.
3. Read "Gaps That Must Be Resolved Before Coding" to understand why more design was necessary.
4. Read "Proposed Architecture" and "Runtime Flows" before creating any files.
5. Use "File-By-File Implementation Guide" as your day-to-day execution checklist.
6. Use "Testing and Validation Strategy" before considering any phase done.

If you only need the shortest version, the imported file is the kernel and this document is the implementation wrapper around it.

## Problem Statement

We want a simple tuple space service with Linda-style semantics. Clients must be able to:

- write tuples into a named space with `out`,
- read a matching tuple without deleting it with `rd`,
- remove exactly one matching tuple with `in`,
- optionally wait for a matching tuple to arrive.

A tuple space is a coordination primitive, not just a key-value store. The core idea is that producers and consumers communicate through structured tuples rather than direct request-response RPC calls. That means the semantics of matching, blocking, visibility after commit, and atomic removal are more important than exposing a large API surface.

The imported plan already establishes the semantic core:

- service methods and candidate HTTP endpoints are defined in the proposal at `import/tuplespace-plan.md:10-32`,
- tuple and template data shapes are defined at `import/tuplespace-plan.md:34-92`,
- the persistence model is defined at `import/tuplespace-plan.md:94-145`,
- matching rules are defined at `import/tuplespace-plan.md:147-183`,
- blocking and concurrency semantics are defined at `import/tuplespace-plan.md:224-358`.

The implementation problem is therefore not "invent tuple space semantics from scratch." The implementation problem is "turn that semantic sketch into a maintainable system with explicit contracts, predictable code layout, operational safety, and a usable CLI."

### Scope

In scope for the first implementation:

- named tuple spaces,
- blocking `rd` and `in`,
- `out`,
- types `string`, `int`, and `bool` in v1,
- HTTP transport,
- Postgres-backed storage and wakeups,
- a Glazed CLI for interacting with the service,
- unit, integration, and concurrency tests.

Explicitly out of scope for v1 unless requirements change:

- full unification,
- stored tuples containing formal fields,
- tuple updates,
- multi-operation transactions across tuple-space calls,
- fair scheduling across waiters,
- durable waiter recovery after process restart,
- gRPC transport.

### Terms

- Tuple: the stored data record, made only of actual values.
- Template: the query pattern, containing actual fields and/or formal fields.
- Actual field: must match by exact type and exact value.
- Formal field: matches by type and binds the tuple value to a variable name.
- Candidate query: SQL that narrows the search space using only actual template fields.
- Semantic match: the final Go-level decision performed by the matcher.

## Proposed Solution

The proposed system has two binaries and six main runtime layers:

1. `tuplespaced`: the server process.
2. `tuplespacectl`: a Glazed-powered client and admin CLI.
3. HTTP API handlers for external requests.
4. Service orchestration for `out`, `rd`, and `in`.
5. Store/repository code for Postgres transactions and candidate selection.
6. Notification fanout for blocked readers and consumers.

The system keeps one very important boundary intact: SQL is for coarse filtering and transactionality; Go is for tuple semantics. This follows the imported plan directly and should not be weakened by trying to move full matching into SQL.

### Evidence Table

The table below separates what is explicit in the imported plan from what this document infers or chooses.

| Source | What It Explicitly Establishes | What This Design Adds |
| --- | --- | --- |
| `import/tuplespace-plan.md:1-8` | Go service, Postgres truth, Go matcher, `LISTEN/NOTIFY` | Operational boundaries, binaries, testing, and config |
| `import/tuplespace-plan.md:10-32` | `TupleSpace` interface and HTTP endpoint shape | Request/response envelopes, error contract, status codes |
| `import/tuplespace-plan.md:34-92` | Tuple/template wire types | Validation rules, v1 type narrowing, serialization guidance |
| `import/tuplespace-plan.md:94-145` | `tuples` and `tuple_fields` schema | migration files, constraints, future indexes, channel naming |
| `import/tuplespace-plan.md:147-183` | Linda-style match semantics | package APIs, binding result type, tests |
| `import/tuplespace-plan.md:186-222` | actual-fields-only candidate query | query builder shape, page size config, repository API |
| `import/tuplespace-plan.md:224-299` | `Out`, `Rd`, `In`, `SKIP LOCKED` | concrete retry loops, timeout behavior, service/store split |
| `import/tuplespace-plan.md:300-328` | long-lived listener and race to avoid | notifier API, subscriber lifecycle, ref counting |
| `import/tuplespace-plan.md:330-387` | package sketch, guarantees, MVP | full repository layout, CLI, playbooks, intern guidance |

## Current-State Analysis

The repository currently contains exactly one imported design note and no implementation code. That matters because every architectural recommendation here must be tied to the imported note, not to an existing codebase. The imported plan is therefore both the source of truth and the main evidence artifact for the ticket.

### Service Boundary And Transport

The proposal defines a `TupleSpace` interface with `Out`, `Rd`, `In`, and optional non-blocking variants `Rdp` and `Inp` in `import/tuplespace-plan.md:12-23`. It also suggests HTTP endpoints:

- `POST /v1/spaces/{space}/out`
- `POST /v1/spaces/{space}/rd`
- `POST /v1/spaces/{space}/in`

Those choices imply three important things:

- the transport should carry a request body for all operations,
- the service boundary is operation-oriented rather than CRUD-oriented,
- the semantic API is small and stable even if the implementation evolves.

### Data Model

The imported plan clearly separates stored tuples from query templates in `import/tuplespace-plan.md:34-92`:

- stored tuples contain actual values only,
- templates may contain actual fields or formal fields,
- field order is significant because matching is positional,
- arity must match exactly.

This is a good first version because it avoids the complexity explosion of full unification while still supporting useful coordination patterns.

### Persistence Model

The imported schema uses:

- `tuples` as canonical storage for the full tuple payload as JSON,
- `tuple_fields` as a normalized lookup table for indexed candidate filtering.

That choice is defined in `import/tuplespace-plan.md:94-145`. The design is strong because it separates:

- payload fidelity, which belongs in `fields_json`,
- query speed, which belongs in typed columns and indexes.

This avoids brittle JSONB expression indexes for every matching case while still making template search practical.

### Matching Semantics

The imported `Match` function in `import/tuplespace-plan.md:147-183` defines the heart of the system:

- arity equality,
- exact type equality,
- actual fields compare exact values,
- formal fields bind values by name.

This means matcher correctness is the semantic core of the project. Every API, SQL query, and concurrency optimization must preserve these rules rather than reinterpret them.

### Candidate Query Strategy

The imported repository strategy in `import/tuplespace-plan.md:186-222` says to build candidate SQL from actual fields only, fetch a bounded row set, decode the payload, and run `Match` in Go. This is the right balance because:

- actual fields are selective and indexable,
- formal fields are not selective enough to justify SQL complexity,
- semantic comparison remains centralized in one Go function.

### Operation Semantics

The operational split is explicit:

- `Out` inserts tuple rows and sends a notification after the insert, inside the transaction flow (`import/tuplespace-plan.md:226-257`),
- `Rd` is non-destructive and can block/retry (`import/tuplespace-plan.md:259-268`),
- `In` is destructive and must atomically delete one tuple after locking candidates with `FOR UPDATE SKIP LOCKED` (`import/tuplespace-plan.md:270-299`).

This is the most important concurrency decision in the entire plan. The use of `SKIP LOCKED` means destructive consumers can race safely without double-consuming the same tuple.

### Blocking And Wakeups

The imported design describes one dedicated `LISTEN/NOTIFY` connection and warns against the race of "scan, then start listening" in `import/tuplespace-plan.md:300-328`. That warning should be treated as a hard invariant. If subscription is not established before the retry loop starts, the service can miss a wakeup and block longer than necessary.

### Package Sketch

The imported plan suggests this package layout:

```text
cmd/tuplesvc/
internal/api/
internal/service/
internal/store/
internal/match/
internal/notify/
internal/types/
```

That is directionally correct, but it is incomplete for a real implementation because it omits:

- migrations,
- config,
- HTTP request/response types,
- observability,
- the requested Glazed CLI.

## Gaps That Must Be Resolved Before Coding

The imported plan is good architecture, but it leaves several implementation-critical questions unanswered.

### 1. HTTP Contract Is Incomplete

The imported note names endpoints but does not define:

- request envelopes,
- response envelopes,
- error codes and error body shape,
- timeout encoding,
- whether bindings should be returned for formal fields.

Without these decisions, the server and CLI cannot be implemented independently.

### 2. Binary Layout Is Unspecified

The imported package sketch implies one service binary only. The user explicitly asked for a Glazed command line utility, so we need either:

- one mixed binary that both serves and acts as client, or
- separate binaries for server and CLI.

This document recommends separate binaries because it gives cleaner dependency boundaries and avoids forcing server-only code to import client output tooling.

### 3. Type Validation Rules Need To Be Explicit

The imported note defines `string`, `int`, `bool`, `float`, and `bytes`, then recommends implementing only `string`, `int`, and `bool` first. We need to make the initial acceptance rules precise:

- reject unsupported field types at the API boundary,
- reject malformed actual/formal combinations,
- reject empty formal names,
- preserve positional field order.

### 4. Notification Channel Naming Needs Hardening

The proposal uses `tuplespace_<space>` as a notification channel name. That is conceptually fine, but the implementation should not pass arbitrary user-controlled space names directly without normalization. The safest approach is to derive a stable, sanitized channel name such as:

```text
tuplespace_<hex(sha1(space))[:16]>
```

and keep the original `space` only as data in the tuple rows.

### 5. Testing Scope Needs To Be Declared Up Front

Tuple spaces are mostly about correctness under concurrency, not just happy-path HTTP behavior. We therefore need explicit tests for:

- semantic matching,
- candidate query generation,
- exactly-once destructive consumption under concurrency,
- blocking wakeups,
- timeout and cancellation behavior,
- CLI request and output formatting.

### 6. Operational Minimums Are Missing

The imported note does not discuss:

- migrations,
- DB pool sizing,
- graceful shutdown,
- logging,
- metrics,
- health checks.

Those are not optional if the goal is an implementation guide rather than a thought experiment.

## Proposed Architecture

### Top-Level Shape

Use two binaries:

```text
cmd/
  tuplespaced/      # server binary
  tuplespacectl/    # Glazed client/admin CLI
internal/
  api/httpapi/
  config/
  match/
  notify/
  service/
  store/
  types/
  validation/
migrations/
```

Why two binaries:

- `tuplespaced` depends on HTTP serving, Postgres, and listener lifecycle.
- `tuplespacectl` depends on HTTP client code and Glazed structured output.
- keeping them separate avoids mixing operational concerns with client presentation logic.

### Component Diagram

```text
                   +----------------------+
                   |   tuplespacectl      |
                   |  Glazed CLI client   |
                   +----------+-----------+
                              |
                              | HTTP JSON
                              v
+----------------------+   +--+-------------------+   +----------------------+
| other clients        |   | tuplespaced         |   | Postgres             |
| curl / tests / apps  +---> httpapi -> service  +---> tuples              |
+----------------------+   |            |        |   | tuple_fields        |
                           |            |        |   | LISTEN / NOTIFY     |
                           |            v        |   +----------------------+
                           |          store      |
                           |            ^        |
                           |            |        |
                           |          notify     |
                           +---------------------+
```

### Recommended Package Layout

```text
cmd/
  tuplespaced/
    main.go
  tuplespacectl/
    main.go
    cmds/
      tuple/
        root.go
        out.go
        rd.go
        in.go
      admin/
        root.go
        health.go
        spaces.go
internal/
  api/httpapi/
    router.go
    handlers.go
    requests.go
    responses.go
    errors.go
  config/
    config.go
  match/
    match.go
    equal.go
    match_test.go
  notify/
    notifier.go
    notifier_test.go
  service/
    service.go
    out.go
    rd.go
    in.go
  store/
    models.go
    candidate_query.go
    tuple_store.go
    tuple_store_test.go
  types/
    tuple.go
    template.go
    bindings.go
  validation/
    tuple.go
    template.go
migrations/
  001_init_tuplespace.sql
```

### API Reference

Choose HTTP first for v1. gRPC can be added later behind the same service interface if needed.

#### `POST /v1/spaces/{space}/out`

Request:

```json
{
  "tuple": {
    "fields": [
      { "type": "string", "value": "job" },
      { "type": "int", "value": 42 }
    ]
  }
}
```

Response:

```json
{
  "ok": true,
  "space": "jobs",
  "arity": 2
}
```

Status code:

- `201 Created` on success

#### `POST /v1/spaces/{space}/rd`

Request:

```json
{
  "template": {
    "fields": [
      { "kind": "actual", "type": "string", "value": "job" },
      { "kind": "formal", "type": "int", "name": "id" }
    ]
  },
  "wait_ms": 30000
}
```

Response:

```json
{
  "ok": true,
  "tuple": {
    "fields": [
      { "type": "string", "value": "job" },
      { "type": "int", "value": 42 }
    ]
  },
  "bindings": {
    "id": 42
  }
}
```

Status codes:

- `200 OK` on match
- `404 Not Found` if no match and `wait_ms == 0`
- `408 Request Timeout` if the deadline expires while waiting

#### `POST /v1/spaces/{space}/in`

Request and response are the same shape as `rd`, but success consumes exactly one tuple.

Status codes:

- `200 OK` on successful removal
- `404 Not Found` if no match and `wait_ms == 0`
- `408 Request Timeout` on wait expiry

#### Error Envelope

Use one error envelope everywhere:

```json
{
  "ok": false,
  "error": {
    "code": "invalid_template",
    "message": "formal field name must be non-empty",
    "details": {}
  }
}
```

Recommended error codes:

- `invalid_space`
- `invalid_tuple`
- `invalid_template`
- `unsupported_type`
- `not_found`
- `timeout`
- `internal`

### Domain Types

Use explicit Go types and keep transport DTOs separate from internal types only if transport requirements diverge later. For v1, it is acceptable for internal and HTTP shapes to be close, but validation must happen before service methods are called.

Recommended internal API:

```go
type ValueType string

const (
    TypeString ValueType = "string"
    TypeInt    ValueType = "int"
    TypeBool   ValueType = "bool"
)

type TupleField struct {
    Type  ValueType `json:"type"`
    Value any       `json:"value"`
}

type Tuple struct {
    Fields []TupleField `json:"fields"`
}

type TemplateFieldKind string

const (
    FieldActual TemplateFieldKind = "actual"
    FieldFormal TemplateFieldKind = "formal"
)

type TemplateField struct {
    Kind  TemplateFieldKind `json:"kind"`
    Type  ValueType         `json:"type"`
    Name  string            `json:"name,omitempty"`
    Value any               `json:"value,omitempty"`
}

type Template struct {
    Fields []TemplateField `json:"fields"`
}

type Bindings map[string]any
```

Validation rules for v1:

- `Tuple.Fields` must be non-empty.
- Every tuple field must use one of `string`, `int`, or `bool`.
- Tuple fields do not have `kind`.
- Template fields must declare `kind`.
- Formal template fields require a non-empty `name` and must omit `value`.
- Actual template fields require `value` and must omit `name`.
- The JSON number handling must normalize to the expected Go type before matching.

### Database Schema And Migrations

Start with the imported schema and make two additions:

1. Add a uniqueness guarantee that each tuple has exactly one row per position through the existing primary key on `(tuple_id, pos)`.
2. Add a check in application code that the number of `tuple_fields` rows equals tuple arity during inserts.

Migration `001_init_tuplespace.sql`:

```sql
CREATE TABLE tuples (
    id          BIGSERIAL PRIMARY KEY,
    space       TEXT NOT NULL,
    arity       INT NOT NULL,
    fields_json JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tuple_fields (
    tuple_id    BIGINT NOT NULL REFERENCES tuples(id) ON DELETE CASCADE,
    pos         INT NOT NULL,
    type        TEXT NOT NULL,
    text_val    TEXT,
    int_val     BIGINT,
    bool_val    BOOLEAN,
    PRIMARY KEY (tuple_id, pos)
);

CREATE INDEX tuples_space_arity_id_idx
    ON tuples(space, arity, id);

CREATE INDEX tuple_fields_text_idx
    ON tuple_fields(pos, type, text_val, tuple_id);

CREATE INDEX tuple_fields_int_idx
    ON tuple_fields(pos, type, int_val, tuple_id);

CREATE INDEX tuple_fields_bool_idx
    ON tuple_fields(pos, type, bool_val, tuple_id);
```

We intentionally drop `float` and `bytea` columns from the first migration because the imported plan recommends `string`, `int`, and `bool` only for the minimal first version in `import/tuplespace-plan.md:360-385`. Add later migrations when those types are actually supported end-to-end.

### Store Layer

The store layer owns SQL, transactions, and row decoding. It must not own tuple semantics beyond basic type mapping.

Recommended store API:

```go
type TupleStore interface {
    InsertTuple(ctx context.Context, tx pgx.Tx, space string, tuple types.Tuple) (int64, error)
    FindCandidates(ctx context.Context, q Querier, space string, tmpl types.Template, limit int) ([]StoredTuple, error)
    LockCandidatesForConsume(ctx context.Context, tx pgx.Tx, space string, tmpl types.Template, limit int) ([]StoredTuple, error)
    DeleteTuple(ctx context.Context, tx pgx.Tx, tupleID int64) error
}
```

`FindCandidates` and `LockCandidatesForConsume` should share a candidate query builder and differ only in lock clauses.

Candidate query pseudocode:

```go
func BuildCandidateQuery(space string, tmpl Template, destructive bool, limit int) SQL {
    sql := NewBuilder()
    sql.Write("SELECT t.id, t.fields_json FROM tuples t")

    aliasIndex := 0
    for pos, field := range tmpl.Fields {
        if field.Kind != FieldActual {
            continue
        }

        alias := fmt.Sprintf("f%d", aliasIndex)
        aliasIndex++
        sql.Write(" JOIN tuple_fields " + alias + " ON " + alias + ".tuple_id = t.id")
        sql.Write(" AND " + alias + ".pos = ?", pos)
        sql.Write(" AND " + alias + ".type = ?", field.Type)
        sql.WriteTypedValueComparison(alias, field)
    }

    sql.Write(" WHERE t.space = ? AND t.arity = ?", space, len(tmpl.Fields))
    sql.Write(" ORDER BY t.id")
    if destructive {
        sql.Write(" FOR UPDATE SKIP LOCKED")
    }
    sql.Write(" LIMIT ?", limit)
    return sql.Build()
}
```

### Matcher Layer

The matcher is the semantic authority. Keep it very small, obvious, and heavily tested.

Recommended function:

```go
func Match(tmpl types.Template, tup types.Tuple) (types.Bindings, bool)
```

Important v1 detail: if the same formal name appears twice in a template, the matcher should require both positions to bind equal values. The imported sketch does not mention repeated formal names, so this is an implementation choice that should be documented and tested.

Matcher pseudocode:

```go
func Match(tmpl Template, tup Tuple) (Bindings, bool) {
    if len(tmpl.Fields) != len(tup.Fields) {
        return nil, false
    }

    bindings := Bindings{}
    for i := range tmpl.Fields {
        tf := tmpl.Fields[i]
        vf := tup.Fields[i]

        if tf.Type != vf.Type {
            return nil, false
        }

        switch tf.Kind {
        case FieldActual:
            if !EqualValue(tf.Type, tf.Value, vf.Value) {
                return nil, false
            }
        case FieldFormal:
            if existing, ok := bindings[tf.Name]; ok && !EqualValue(tf.Type, existing, vf.Value) {
                return nil, false
            }
            bindings[tf.Name] = vf.Value
        default:
            return nil, false
        }
    }

    return bindings, true
}
```

### Notify Layer

The notify layer exists to avoid polling loops and to centralize the blocking race handling identified in `import/tuplespace-plan.md:314-327`.

Recommended interface:

```go
type Notifier interface {
    Subscribe(space string) (Subscription, error)
    NotifyChannel(space string) string
    Close() error
}

type Subscription interface {
    C() <-chan struct{}
    Close() error
}
```

Design rules:

- one dedicated Postgres connection per service instance,
- subscribe before the first scan attempt,
- track subscribers per space in memory,
- fan out one lightweight wakeup signal to all subscribers,
- use non-blocking sends so a slow waiter does not stall the notifier,
- treat notifications as hints, not proofs of availability.

Why notifications are hints:

- a notification can wake multiple waiters,
- another consumer may win the race before a given waiter retries,
- therefore every wakeup must re-run the query and matcher.

### Service Layer

The service layer owns operation semantics. It composes store, matcher, notifier, and timeout handling.

Recommended service struct:

```go
type Service struct {
    db             *pgxpool.Pool
    store          store.TupleStore
    notifier       notify.Notifier
    candidateLimit int
}
```

Recommended interface:

```go
type TupleSpace interface {
    Out(ctx context.Context, space string, tuple types.Tuple) error
    Rd(ctx context.Context, space string, tmpl types.Template, wait time.Duration) (types.Tuple, types.Bindings, error)
    In(ctx context.Context, space string, tmpl types.Template, wait time.Duration) (types.Tuple, types.Bindings, error)
    Rdp(ctx context.Context, space string, tmpl types.Template) (types.Tuple, types.Bindings, bool, error)
    Inp(ctx context.Context, space string, tmpl types.Template) (types.Tuple, types.Bindings, bool, error)
}
```

Why return bindings from `Rd` and `In`:

- templates with formal fields are more useful if the caller receives the extracted bindings,
- the imported plan's `Match` function already produces an environment map,
- the CLI can render those bindings directly.

### Glazed CLI Design

The user explicitly requested a Glazed command line utility, so this is not optional. The local Glazed tutorial shows the expected construction pattern:

- implement `RunIntoGlazeProcessor` for structured output in `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md:103-151`,
- construct the command with `cmds.NewCommandDescription`, `fields.New`, `settings.NewGlazedSchema`, and `cli.NewCommandSettingsSection` in `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md:153-238`,
- build Cobra commands with `cli.BuildCobraCommandFromCommand` from `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cobra.go:379-407`,
- rely on `cli.NewCommandSettingsSection` for `--print-yaml`, `--print-parsed-fields`, and `--print-schema` as defined in `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cli.go:86-117`.

Recommended CLI tree:

```text
tuplespacectl tuple out
tuplespacectl tuple rd
tuplespacectl tuple in
tuplespacectl admin health
tuplespacectl admin spaces
```

Recommended Glazed group layout:

```text
cmd/tuplespacectl/
  main.go
  cmds/
    tuple/
      root.go
      out.go
      rd.go
      in.go
    admin/
      root.go
      health.go
      spaces.go
```

Suggested CLI behavior:

- `tuple out` accepts either repeated `--field` values or a JSON tuple file.
- `tuple rd` and `tuple in` accept a JSON template file or repeated field specifications.
- the default output is a readable table, but `--output json` should expose the raw tuple and bindings for scripting.

Glazed command skeleton:

```go
type InCommand struct {
    *cmds.CommandDescription
    client *client.HTTPClient
}

type InSettings struct {
    Space    string `glazed:"space"`
    Template string `glazed:"template-file"`
    WaitMs   int    `glazed:"wait-ms"`
}

func NewInCommand(client *client.HTTPClient) (*InCommand, error) {
    glazedSection, err := settings.NewGlazedSchema()
    if err != nil {
        return nil, err
    }

    commandSettingsSection, err := cli.NewCommandSettingsSection()
    if err != nil {
        return nil, err
    }

    cmdDesc := cmds.NewCommandDescription(
        "in",
        cmds.WithShort("Consume one matching tuple"),
        cmds.WithFlags(
            fields.New("space", fields.TypeString, fields.WithHelp("Tuple space name")),
            fields.New("template-file", fields.TypeString, fields.WithHelp("Path to template JSON file")),
            fields.New("wait-ms", fields.TypeInteger, fields.WithDefault(0), fields.WithHelp("How long to wait")),
        ),
        cmds.WithSectionsList(glazedSection, commandSettingsSection),
    )

    return &InCommand{CommandDescription: cmdDesc, client: client}, nil
}

func (c *InCommand) RunIntoGlazeProcessor(
    ctx context.Context,
    vals *values.Values,
    gp middlewares.Processor,
) error {
    settings := &InSettings{}
    if err := vals.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
        return err
    }

    resp, err := c.client.In(ctx, settings.Space, settings.Template, settings.WaitMs)
    if err != nil {
        return err
    }

    row := types.NewRow(
        types.MRP("space", settings.Space),
        types.MRP("tuple", resp.Tuple),
        types.MRP("bindings", resp.Bindings),
    )
    return gp.AddRow(ctx, row)
}
```

### Configuration

Recommended config fields:

```go
type Config struct {
    HTTPListenAddr string
    DatabaseURL    string
    CandidateLimit int
    ShutdownGrace  time.Duration
}
```

Environment variable mapping:

- `TUPLESPACE_HTTP_LISTEN_ADDR`
- `TUPLESPACE_DATABASE_URL`
- `TUPLESPACE_CANDIDATE_LIMIT`
- `TUPLESPACE_SHUTDOWN_GRACE`

Defaults:

- listen address: `:8080`
- candidate limit: `64`
- shutdown grace: `10s`

## Design Decisions

### Decision 1: HTTP First, Not gRPC First

Reasoning:

- the imported note allows HTTP or gRPC,
- the repo currently has no implementation,
- the requested Glazed CLI will talk to HTTP easily,
- HTTP keeps onboarding and debugging simple with `curl`.

Tradeoff:

- gRPC would give stronger schema tooling later, but it increases v1 surface area and is not required for semantic correctness.

### Decision 2: Separate Server And CLI Binaries

Reasoning:

- keeps dependency boundaries clean,
- lets the CLI evolve as an operator tool,
- avoids mixing service startup with client output concerns.

Tradeoff:

- two binaries are slightly more setup work than one.

### Decision 3: Keep Semantic Matching In Go

Reasoning:

- explicitly required by the imported note,
- easier to test,
- avoids encoding Linda semantics in SQL join logic.

Tradeoff:

- more tuples may be decoded after candidate selection, but the SQL filter limits this.

### Decision 4: Use `FOR UPDATE SKIP LOCKED` For `In`

Reasoning:

- directly recommended by the imported note,
- gives exactly-once destructive consumption under concurrency,
- avoids serializing all consumers on one hot tuple.

Tradeoff:

- tuple selection is not fair or deterministic across competing consumers.

### Decision 5: Restrict V1 Types To `string`, `int`, `bool`

Reasoning:

- explicitly recommended by the imported note as the minimal viable build,
- simplifies validation and JSON number normalization,
- reduces migration and matcher complexity.

Tradeoff:

- future type expansion requires migrations and more tests.

The five decisions above should be treated as architecture-level choices, not implementation accidents. They constrain file layout, tests, and API contracts. If one of them changes later, the change should be recorded as a new design update rather than silently drifting in implementation.

## Alternatives Considered

### Full SQL Matching

Rejected for v1 because the imported plan intentionally keeps semantic matching in Go. Pushing all formal/actual logic into SQL would make the query generator harder to reason about and harder to test.

### Polling Instead Of `LISTEN/NOTIFY`

Rejected because the imported plan already calls for `LISTEN/NOTIFY`, and polling would either:

- add avoidable database load, or
- weaken responsiveness for blocked readers.

### Single JSONB Table Without `tuple_fields`

Rejected because the imported plan already uses normalized lookup rows for indexing. A JSONB-only design would make candidate selection slower and more fragile as template complexity grows.

### One Unified Binary For Server And CLI

Rejected because the Glazed CLI is better treated as an operator and debugging client. Keeping it separate produces a clearer architecture and reduces incidental dependencies in the service runtime.

### gRPC-First API

Rejected for v1 because nothing in the imported note requires it, and the user explicitly asked for a Glazed command line utility, which pairs naturally with HTTP JSON.

## Implementation Plan

This section turns the architecture into an execution plan. Read it when you are ready to translate the design into files, tests, and runtime behavior.

### Runtime Flows

### `Out` Sequence

```text
Client -> HTTP handler -> validation -> service.Out
service.Out -> begin tx
service.Out -> store.InsertTuple
service.Out -> store.InsertTupleFields
service.Out -> SELECT pg_notify(channel, '')
service.Out -> commit
service.Out -> return success
```

`Out` pseudocode:

```go
func (s *Service) Out(ctx context.Context, space string, tuple types.Tuple) error {
    if err := validation.ValidateSpace(space); err != nil {
        return err
    }
    if err := validation.ValidateTuple(tuple); err != nil {
        return err
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    tupleID, err := s.store.InsertTuple(ctx, tx, space, tuple)
    if err != nil {
        return err
    }
    if err := s.store.InsertTupleFields(ctx, tx, tupleID, tuple); err != nil {
        return err
    }
    if _, err := tx.Exec(ctx, "SELECT pg_notify($1, '')", s.notifier.NotifyChannel(space)); err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### `Rd` Sequence

```text
subscribe(space)
loop:
  candidate rows = store.FindCandidates(...)
  for each row:
    if matcher accepts row:
      return tuple
  if deadline reached:
    return timeout
  wait for notify or context cancel
```

`Rd` pseudocode:

```go
func (s *Service) Rd(ctx context.Context, space string, tmpl types.Template, wait time.Duration) (types.Tuple, types.Bindings, error) {
    sub, err := s.notifier.Subscribe(space)
    if err != nil {
        return types.Tuple{}, nil, err
    }
    defer sub.Close()

    deadlineCtx := withOptionalTimeout(ctx, wait)

    for {
        rows, err := s.store.FindCandidates(deadlineCtx, s.db, space, tmpl, s.candidateLimit)
        if err != nil {
            return types.Tuple{}, nil, err
        }

        for _, row := range rows {
            bindings, ok := match.Match(tmpl, row.Tuple)
            if ok {
                return row.Tuple, bindings, nil
            }
        }

        if wait == 0 {
            return types.Tuple{}, nil, ErrNotFound
        }

        select {
        case <-deadlineCtx.Done():
            return types.Tuple{}, nil, MapDeadlineErr(deadlineCtx.Err())
        case <-sub.C():
        }
    }
}
```

### `In` Sequence

```text
subscribe(space)
loop:
  begin tx
  select candidate rows FOR UPDATE SKIP LOCKED
  for each row:
    if matcher accepts row:
      delete row
      commit
      return tuple
  rollback
  if deadline reached:
    return timeout
  wait for notify or context cancel
```

Sequence diagram:

```text
Client         Handler        Service         Store            Postgres        Notifier
  |               |              |               |                 |              |
  | POST /in      |              |               |                 |              |
  |-------------->|              |               |                 |              |
  |               | validate     |               |                 |              |
  |               |------------->| subscribe     |                 |              |
  |               |              |-------------------------------->| LISTEN       |
  |               |              | begin tx      |                 |              |
  |               |              |-------------->| query candidates|              |
  |               |              |               |---------------->| lock rows     |
  |               |              | match in Go   |                 |              |
  |               |              | delete tuple  |                 |              |
  |               |              |-------------->| DELETE          |              |
  |               |              |               |---------------->|              |
  |               |              | commit        |                 |              |
  |               |              |-------------------------------->| COMMIT       |
  |               | <------------| tuple+bindings|                 |              |
  | <-------------|              |               |                 |              |
```

### File-By-File Implementation Guide

### Server Entry Point

`cmd/tuplespaced/main.go`

- parse config,
- initialize logger,
- open `pgxpool`,
- initialize notifier,
- initialize store and service,
- start HTTP server,
- handle graceful shutdown.

### CLI Entry Point

`cmd/tuplespacectl/main.go`

- build the root Cobra command,
- register Glazed tuple and admin subcommands,
- add logging if desired,
- initialize an HTTP client shared by subcommands.

### Domain Types

`internal/types/tuple.go`

- define `ValueType`, `TupleField`, and `Tuple`.

`internal/types/template.go`

- define `TemplateFieldKind`, `TemplateField`, and `Template`.

`internal/types/bindings.go`

- define `Bindings`.

### Validation

`internal/validation/tuple.go`

- validate tuple field count, supported types, and actual values.

`internal/validation/template.go`

- validate `kind`, `name`, type support, and field shape.

### Matcher

`internal/match/match.go`

- implement `Match`.

`internal/match/equal.go`

- implement type-specific equality normalization.

`internal/match/match_test.go`

- exhaustive semantic tests.

### Store

`internal/store/models.go`

- define decoded store records such as `StoredTuple`.

`internal/store/candidate_query.go`

- build SQL for actual-only candidate filtering.

`internal/store/tuple_store.go`

- execute inserts, candidate queries, lock queries, and deletes.

`internal/store/tuple_store_test.go`

- integration tests against Postgres.

### Notifier

`internal/notify/notifier.go`

- manage one listener connection,
- subscribe/unsubscribe per space,
- fan out notifications.

`internal/notify/notifier_test.go`

- verify subscribe-before-scan safety and wakeup fanout behavior.

### Service

`internal/service/service.go`

- assemble dependencies and shared helpers.

`internal/service/out.go`

- implement `Out`.

`internal/service/rd.go`

- implement `Rd` and `Rdp`.

`internal/service/in.go`

- implement `In` and `Inp`.

### HTTP API

`internal/api/httpapi/requests.go`

- define request DTOs.

`internal/api/httpapi/responses.go`

- define response DTOs and error envelope.

`internal/api/httpapi/errors.go`

- map domain errors to HTTP status codes.

`internal/api/httpapi/handlers.go`

- implement handlers for `out`, `rd`, and `in`.

`internal/api/httpapi/router.go`

- wire routes and health endpoints.

### Database Migrations

`migrations/001_init_tuplespace.sql`

- create `tuples`,
- create `tuple_fields`,
- add indexes.

### Glazed CLI Commands

`cmd/tuplespacectl/cmds/tuple/out.go`

- parse tuple input,
- call `POST /out`,
- emit confirmation row.

`cmd/tuplespacectl/cmds/tuple/rd.go`

- call `POST /rd`,
- emit tuple and bindings.

`cmd/tuplespacectl/cmds/tuple/in.go`

- call `POST /in`,
- emit consumed tuple and bindings.

`cmd/tuplespacectl/cmds/admin/health.go`

- call `/healthz`,
- emit service health row.

### Phased Implementation Plan

### Phase 0: Repository Bootstrap

1. Create module and binary layout.
2. Add config loading.
3. Add migration runner or migration instructions.
4. Add a minimal `/healthz` endpoint.

Done when:

- service starts,
- Postgres connects,
- migrations apply,
- health endpoint responds.

### Phase 1: Core Types, Validation, And Matcher

1. Implement `internal/types`.
2. Implement `internal/validation`.
3. Implement `internal/match`.
4. Add exhaustive matcher tests.

Done when:

- tuples and templates validate correctly,
- repeated formal name behavior is tested,
- equality is type-safe.

### Phase 2: Store Layer

1. Implement migrations.
2. Implement tuple inserts.
3. Implement candidate query builder.
4. Implement read and destructive lock queries.
5. Add store integration tests.

Done when:

- tuples persist correctly,
- candidate queries return bounded matches,
- destructive locks do not double-select the same row in one test run.

### Phase 3: Notifier And Blocking Semantics

1. Implement notifier subscription/fanout.
2. Wire `LISTEN/NOTIFY`.
3. Add tests for wakeups and cancellation.

Done when:

- blocked readers wake on `out`,
- cancellation unblocks waits,
- missed-wakeup race is covered by tests.

### Phase 4: Service Operations

1. Implement `Out`.
2. Implement `Rd` and `Rdp`.
3. Implement `In` and `Inp`.
4. Add concurrency tests.

Done when:

- `rd` is non-destructive,
- `in` is destructive and exactly-once,
- timeout behavior is correct.

### Phase 5: HTTP API

1. Implement request and response DTOs.
2. Implement handlers and router.
3. Add HTTP integration tests.

Done when:

- each endpoint returns the documented status codes,
- invalid input maps to stable error codes,
- bindings are included in `rd` and `in` responses.

### Phase 6: Glazed CLI

1. Implement HTTP client package for the CLI.
2. Implement Glazed tuple commands.
3. Implement admin commands.
4. Add CLI smoke tests or scripted examples.

Done when:

- operators can `out`, `rd`, and `in` from the shell,
- `--output json` works for automation,
- help output is clear and examples run.

## Testing And Validation Strategy

### Unit Tests

Target:

- `internal/match`,
- `internal/validation`,
- candidate query builder,
- HTTP error mapping.

Must cover:

- exact type mismatch,
- actual value mismatch,
- repeated formal names,
- empty or invalid template fields,
- unsupported types,
- zero-wait `rd` and `in` not-found behavior.

### Integration Tests

Use Postgres in tests.

Target:

- inserts into both tables,
- candidate query performance path,
- delete semantics,
- listener wakeups.

### Concurrency Tests

These are mandatory.

Test cases:

1. two concurrent `in` callers for one matching tuple result in exactly one success,
2. two concurrent `rd` callers can both observe the same tuple,
3. blocked `in` wakes after `out`,
4. timeout cancels waiting loops cleanly.

### API Tests

Test request validation, response envelopes, and status codes for:

- success,
- invalid tuple/template,
- unsupported type,
- not found,
- timeout.

### CLI Validation

Smoke commands for a local stack:

```bash
tuplespacectl tuple out --space jobs --tuple-file ./examples/job-42.json
tuplespacectl tuple rd --space jobs --template-file ./examples/job-any.json --wait-ms 1000
tuplespacectl tuple in --space jobs --template-file ./examples/job-any.json --wait-ms 1000 --output json
```

## Risks, Alternatives, And Open Questions

### Risks

- JSON number normalization can accidentally blur `int` semantics if decoding is not strict.
- Notification storms may wake more waiters than necessary under hot spaces.
- Long waits require careful context handling to avoid goroutine leaks.
- Candidate queries may become under-selective if templates contain few actual fields.

### Open Questions

1. Should v1 expose non-blocking `Rdp` and `Inp` over HTTP immediately, or keep them as internal service methods until there is a concrete client need?
2. Should repeated formal names be legal and equality-constrained, or rejected as ambiguous? This document recommends allowing them with equality checks.
3. Should the CLI support inline field syntax, JSON files, or both in v1? This document recommends supporting JSON files first because they are less ambiguous.

### Operational Notes

- Do not promise fairness. The imported plan explicitly does not.
- Do not promise durable waiters across restart. The imported plan explicitly does not.
- Do not promise deterministic tuple choice among multiple matches. Always document first-match-by-query-order behavior instead.

The main unresolved questions are listed above. None of them block initial implementation, but they should be acknowledged in code comments and test names so future contributors understand which behaviors are intentionally chosen and which remain policy choices.

## References

- Imported source architecture note:
  - `/home/manuel/code/wesen/2026-03-22--tuplespace/import/tuplespace-plan.md`
- Glazed tutorial for command shape:
  - `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/doc/tutorials/05-build-first-command.md`
- Glazed Cobra command builder:
  - `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cobra.go`
- Glazed command settings section:
  - `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cli/cli.go`
- Glazed command description constructor:
  - `/home/manuel/code/wesen/corporate-headquarters/glazed/pkg/cmds/cmds.go`
