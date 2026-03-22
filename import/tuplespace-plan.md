A simple design is:

* **Go service** exposes `out`, `rd`, `in` over HTTP or gRPC
* **Postgres** is the source of truth for tuples
* **Go matcher** implements Linda-style template matching
* **Postgres LISTEN/NOTIFY** wakes blocked `rd`/`in` calls when new tuples arrive

This keeps the semantics simple and puts all concurrency control in Postgres.

## 1. Service boundary

Define the core interface in Go first:

```go
type TupleSpace interface {
    Out(ctx context.Context, space string, tuple Tuple) error
    Rd(ctx context.Context, space string, tmpl Template, wait time.Duration) (Tuple, error)
    In(ctx context.Context, space string, tmpl Template, wait time.Duration) (Tuple, error)

    // optional non-blocking variants
    Rdp(ctx context.Context, space string, tmpl Template) (Tuple, bool, error)
    Inp(ctx context.Context, space string, tmpl Template) (Tuple, bool, error)
}
```

Suggested transport:

* `POST /v1/spaces/{space}/out`
* `POST /v1/spaces/{space}/rd`
* `POST /v1/spaces/{space}/in`

Use `POST` for all three because `rd` and `in` need a template in the request body.

## 2. Data model

Assume **stored tuples contain only actual fields**. Templates can contain actual or formal fields.

### Wire / API representation

```json
{
  "fields": [
    { "kind": "actual", "type": "string", "value": "job" },
    { "kind": "actual", "type": "int",    "value": 42 }
  ]
}
```

Template:

```json
{
  "fields": [
    { "kind": "actual", "type": "string", "value": "job" },
    { "kind": "formal", "type": "int",    "name": "id" }
  ]
}
```

### Go types

```go
type ValueType string

const (
    TypeString ValueType = "string"
    TypeInt    ValueType = "int"
    TypeBool   ValueType = "bool"
    TypeFloat  ValueType = "float"
    TypeBytes  ValueType = "bytes"
)

type TupleField struct {
    Type  ValueType `json:"type"`
    Value any       `json:"value"`
}

type Tuple struct {
    Fields []TupleField `json:"fields"`
}

type TemplateField struct {
    Kind  string    `json:"kind"` // actual | formal
    Type  ValueType `json:"type"`
    Name  string    `json:"name,omitempty"`
    Value any       `json:"value,omitempty"`
}

type Template struct {
    Fields []TemplateField `json:"fields"`
}
```

## 3. Postgres schema

Use one table for tuples and one table for indexed fields.

```sql
CREATE TABLE tuples (
    id          BIGSERIAL PRIMARY KEY,
    space       TEXT NOT NULL,
    arity       INT  NOT NULL,
    fields_json JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tuple_fields (
    tuple_id    BIGINT NOT NULL REFERENCES tuples(id) ON DELETE CASCADE,
    pos         INT    NOT NULL,
    type        TEXT   NOT NULL,

    text_val    TEXT,
    int_val     BIGINT,
    bool_val    BOOLEAN,
    float_val   DOUBLE PRECISION,
    bytea_val   BYTEA,

    PRIMARY KEY (tuple_id, pos)
);
```

Indexes:

```sql
CREATE INDEX tuples_space_arity_id_idx
    ON tuples(space, arity, id);

CREATE INDEX tuple_fields_text_idx
    ON tuple_fields(pos, type, text_val, tuple_id);

CREATE INDEX tuple_fields_int_idx
    ON tuple_fields(pos, type, int_val, tuple_id);

CREATE INDEX tuple_fields_bool_idx
    ON tuple_fields(pos, type, bool_val, tuple_id);

CREATE INDEX tuple_fields_float_idx
    ON tuple_fields(pos, type, float_val, tuple_id);
```

Why this shape:

* `tuples.fields_json` is the canonical tuple payload
* `tuple_fields` exists only to narrow candidate search efficiently
* matching still happens in Go, not in SQL

## 4. Matching model

Do not implement full unification. Implement Linda-style matching:

* arity must match
* actual template field matches tuple field by exact type and exact value
* formal template field matches tuple field by type and binds the value
* stored tuples are actual-only

```go
func Match(tmpl Template, tup Tuple) (map[string]any, bool) {
    if len(tmpl.Fields) != len(tup.Fields) {
        return nil, false
    }

    env := map[string]any{}
    for i := range tmpl.Fields {
        tf := tmpl.Fields[i]
        vf := tup.Fields[i]

        if tf.Type != vf.Type {
            return nil, false
        }

        switch tf.Kind {
        case "actual":
            if !equalValue(tf.Value, vf.Value) {
                return nil, false
            }
        case "formal":
            env[tf.Name] = vf.Value
        default:
            return nil, false
        }
    }
    return env, true
}
```

## 5. Repository strategy

For a template, generate a **candidate SQL query** from its actual fields only.

Example template:

```text
("job", ?int, true)
```

Candidate SQL:

```sql
SELECT t.id, t.fields_json
FROM tuples t
JOIN tuple_fields f0
  ON f0.tuple_id = t.id
 AND f0.pos = 0
 AND f0.type = 'string'
 AND f0.text_val = 'job'
JOIN tuple_fields f2
  ON f2.tuple_id = t.id
 AND f2.pos = 2
 AND f2.type = 'bool'
 AND f2.bool_val = true
WHERE t.space = $1
  AND t.arity = 3
ORDER BY t.id
LIMIT 64;
```

Then decode those rows and run `Match(...)` in Go.

This split is useful:

* SQL does coarse filtering
* Go does exact semantic checking

## 6. Operation semantics

### `Out`

`Out` is straightforward:

1. begin tx
2. insert row into `tuples`
3. insert one row per field into `tuple_fields`
4. `NOTIFY tuplespace_<space>`
5. commit

Pseudo-code:

```go
func (s *Service) Out(ctx context.Context, space string, tup Tuple) error {
    tx, err := s.db.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    tupleID, err := insertTuple(ctx, tx, space, tup)
    if err != nil { return err }

    if err := insertTupleFields(ctx, tx, tupleID, tup); err != nil {
        return err
    }

    if _, err := tx.Exec(ctx, `SELECT pg_notify($1, '')`, notifyChannel(space)); err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### `Rd`

`Rd` does not lock or delete.

Loop:

1. run candidate query
2. match in Go
3. return first match if found
4. if none and wait allowed, block on notification or timeout, then retry

### `In`

`In` must remove exactly one tuple atomically.

Loop:

1. begin tx
2. select candidates with `FOR UPDATE SKIP LOCKED`
3. match in Go
4. `DELETE FROM tuples WHERE id = $1`
5. commit and return

Pseudo-SQL:

```sql
SELECT t.id, t.fields_json
FROM ...
WHERE t.space = $1
  AND t.arity = $2
ORDER BY t.id
FOR UPDATE SKIP LOCKED
LIMIT 64;
```

Why `SKIP LOCKED`:

* concurrent `in` callers do not block each other
* two consumers cannot remove the same tuple
* `rd` remains non-destructive

## 7. Blocking implementation

Use one dedicated Postgres connection per service instance for `LISTEN/NOTIFY`.

Architecture:

* `Notifier` goroutine does `LISTEN tuplespace_<space>`
* it fans out notifications to in-memory subscribers
* blocked `rd`/`in` calls subscribe, attempt scan, and wait for either:

  * a notification
  * context cancellation
  * timeout

Important race to avoid:

* do not do “scan, then start listening”
* either keep the listener active for the lifetime of the process, or ensure subscription is established before the retry loop starts

A simple pattern is:

```text
subscribe(space)
for {
    tuple := tryImmediate()
    if found: return
    wait for notify or deadline
}
```

## 8. Go package layout

```text
cmd/tuplesvc/
internal/api/          // HTTP handlers or gRPC
internal/service/      // Out/Rd/In orchestration
internal/store/        // SQL repo
internal/match/        // Linda matcher
internal/notify/       // LISTEN/NOTIFY fanout
internal/types/        // Tuple, Template, Field
```

## 9. Concurrency and guarantees

What this design guarantees:

* `out` is visible only after commit
* `in` removes at most one tuple exactly once
* concurrent `in` consumers do not double-consume
* `rd` may return the same tuple to multiple readers
* blocked operations wake up on new tuple arrival

What it does not guarantee:

* strict fairness
* deterministic choice among multiple matches
* durable waiter state across service restarts

For a simple service, those omissions are acceptable.

## 10. Minimal first version

If you want the smallest viable build, implement only:

* `out`
* blocking `rd`
* blocking `in`
* `string`, `int`, `bool`

Skip:

* inverse matching with formal fields inside stored tuples
* tuple updates
* transactions across multiple tuple-space ops
* persistent waiter queues

## 11. Practical recommendation

Start with this exact stack:

* `pgx/v5` + `pgxpool`
* JSON payload as canonical tuple storage
* normalized `tuple_fields` for indexing
* `FOR UPDATE SKIP LOCKED` for `in`
* `LISTEN/NOTIFY` for wakeups
* matching in Go, not SQL

That is simple, correct enough, and easy to evolve.
  
