---
Title: TupleSpace Server Overview
Slug: tuplespaced-overview
Short: Learn what tuplespaced does, what it depends on, and which runtime flags matter.
Topics:
- tuplespace
- server
Commands:
- tuplespaced
Flags:
- http-listen-addr
- database-url
- candidate-limit
- shutdown-grace
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`tuplespaced` is the TupleSpace HTTP server. It owns the runtime loop, the Postgres-backed notifier, the migration application step, and the HTTP API that `tuplespacectl` talks to.

## Startup Sequence

At startup, the server performs four critical actions:

1. validate runtime configuration
2. connect to Postgres
3. apply migrations from the local `migrations/` directory
4. start the HTTP API and notifier-backed coordination paths

If any of those steps fail, the process exits instead of starting in a degraded state.

## Important Flags

- `--http-listen-addr` controls the bind address for the HTTP API
- `--database-url` selects the Postgres database
- `--candidate-limit` caps the number of candidate tuples considered in scans
- `--shutdown-grace` controls graceful shutdown timing

## Operational Notes

The server uses structured logging and exposes admin endpoints for inspection. In a local development setup, it is typically started through `docker compose`, but the binary can also be run directly for debugging.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| Startup fails before listening | Database connection or migration setup failed | Check `--database-url` and whether Postgres is reachable |
| Server exits on startup | Configuration is invalid | Verify the flag values and duration formats |
| CLI can reach `/healthz` but not admin routes | An older server binary or image is still running | Rebuild and restart the active server process |

## See Also

- `tuplespaced help tuplespaced-local-dev`
