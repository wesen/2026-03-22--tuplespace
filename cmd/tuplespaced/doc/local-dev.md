---
Title: Running tuplespaced Locally
Slug: tuplespaced-local-dev
Short: Start the TupleSpace server locally and point tuplespacectl at it.
Topics:
- tuplespace
- server
- local-development
Commands:
- tuplespaced
- tuplespacectl
Flags:
- http-listen-addr
- database-url
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide covers the local development loop for the server. It assumes you want a direct process or Docker-based workflow with Postgres available on your machine.

## Step 1: Start Postgres

The repository includes a `docker compose` setup for Postgres and the server. For a Postgres-only loop, start the database first.

```bash
docker compose up -d postgres
```

## Step 2: Run the Server

Start the server with an explicit database URL.

```bash
go run ./cmd/tuplespaced --database-url postgres://postgres:postgres@127.0.0.1:15433/tuplespace?sslmode=disable
```

## Step 3: Probe It From the CLI

Use the client binary to confirm the server is reachable.

```bash
go run ./cmd/tuplespacectl admin health --server-url http://127.0.0.1:8080
```

## Step 4: Inspect Logs and State

Use structured logs from the server and admin commands from the CLI together. This is the fastest way to debug route mismatches, notifier behavior, and tuple lifecycle issues.

## See Also

- `tuplespaced help tuplespaced-overview`
