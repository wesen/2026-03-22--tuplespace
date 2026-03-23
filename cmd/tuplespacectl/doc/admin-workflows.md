---
Title: Admin Workflows
Slug: admin-workflows
Short: Overview of the read-only and maintenance-oriented admin commands.
Topics:
- tuplespace
- admin
Commands:
- admin
Flags:
- space
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

The `admin` command group is for operators, debugging, and maintenance. It complements the Linda tuple operations by exposing the raw state of the system and the server runtime.

## Read-Only Inspection

Use these commands when you need to understand the current state without changing data:

- `admin health` verifies the server is reachable
- `admin spaces` lists known spaces and tuple counts
- `admin stats` shows runtime counters and notifier metrics
- `admin dump` prints tuples from one space or all spaces
- `admin peek` retrieves a filtered view of tuples
- `admin export` emits tuples in machine-readable form
- `admin config` shows the server runtime configuration snapshot
- `admin schema` shows migration state
- `admin waiters` shows currently blocked operations

## Maintenance Commands

Use these when you need to change data or exercise control paths:

- `admin tuple get` fetches a tuple by internal id
- `admin tuple delete` removes a tuple by internal id
- `admin purge` removes tuples from a space, usually with explicit confirmation
- `admin notify-test` exercises the notifier plumbing for diagnostics

Because these commands are operationally sensitive, start with read-only inspection before deleting or purging data.

## Example Session

```bash
tuplespacectl admin spaces
tuplespacectl admin dump --space jobs --output json
tuplespacectl admin stats --output yaml
```

## See Also

- `tuplespacectl help tuplespacectl-overview`
- `tuplespacectl help tutorial-first-roundtrip`
