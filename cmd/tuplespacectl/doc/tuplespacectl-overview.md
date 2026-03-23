---
Title: TupleSpace CLI Overview
Slug: tuplespacectl-overview
Short: Learn how tuplespacectl is organized and how to approach tuple and admin commands.
Topics:
- tuplespace
- cli
Commands:
- tuplespacectl
- tuple
- admin
Flags:
- server-url
- space
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`tuplespacectl` is the operator and client-facing entry point for the TupleSpace system. It talks to `tuplespaced` over HTTP and exposes two main command groups: `tuple` for Linda-style tuple operations and `admin` for inspection and maintenance workflows.

Use `tuple` when you want to write, read, or consume tuples from a named space. Use `admin` when you need to inspect spaces, dump raw tuple contents, view runtime stats, or perform targeted maintenance such as tuple deletion or space purges.

## Command Layout

- `tuplespacectl tuple out` writes one or more tuples.
- `tuplespacectl tuple rd` reads matching tuples without consuming them.
- `tuplespacectl tuple in` reads and consumes matching tuples.
- `tuplespacectl admin ...` exposes health, stats, dump, export, schema, and maintenance commands.

## Runtime Defaults

Most commands share two important flags:

- `--server-url` selects the TupleSpace server base URL.
- `--space` selects the logical tuple space for tuple operations and some admin queries.

The CLI also supports environment-driven defaults through Glazed parsing. In practice, `TUPLESPACECTL_SERVER_URL` and `TUPLESPACECTL_SPACE` let you avoid repeating those flags on every command.

## Input Formats

Tuple commands accept either JSON files or the compact tuple DSL. The DSL is the fastest way to learn the tool because it stays close to the tuple structure while remaining shell-friendly.

Examples:

```bash
tuplespacectl tuple out --space jobs 'job,42,true'
tuplespacectl tuple rd --space jobs 'job,?id:int,?ready:bool'
tuplespacectl admin dump --space jobs --output json
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `connection refused` | `tuplespaced` is not running or the URL is wrong | Check `--server-url` or `TUPLESPACECTL_SERVER_URL`, then run `tuplespacectl admin health` |
| `not_found` from `rd` or `in` | No tuple matches the template | Confirm the space name, tuple arity, and field types |
| `decode error response` or unexpected HTTP error | Server and CLI versions are out of sync, or the server returned a plain text error | Rebuild/restart the server and retry the request |

## See Also

- `tuplespacectl help tuple-dsl`
- `tuplespacectl help admin-workflows`
