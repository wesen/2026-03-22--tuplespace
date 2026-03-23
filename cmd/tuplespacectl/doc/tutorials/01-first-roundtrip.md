---
Title: First Tuple Roundtrip
Slug: tutorial-first-roundtrip
Short: Write, read, and consume a tuple with tuplespacectl.
Topics:
- tuplespace
- tutorial
Commands:
- tuple
- admin
Flags:
- server-url
- space
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This walkthrough gives you the shortest useful path through the CLI. By the end, you will have written a tuple, inspected the server, read the tuple without consuming it, and then consumed it.

## Prerequisites

- `tuplespaced` is running
- `tuplespacectl` can reach the server URL you plan to use

## Step 1: Confirm Connectivity

Start by checking that the server is reachable.

```bash
tuplespacectl admin health --server-url http://127.0.0.1:8080
```

## Step 2: Write a Tuple

Write a simple tuple into the `jobs` space.

```bash
tuplespacectl tuple out --server-url http://127.0.0.1:8080 --space jobs 'job,42,true'
```

## Step 3: Inspect the Space

Use an admin command to confirm the tuple is present.

```bash
tuplespacectl admin dump --server-url http://127.0.0.1:8080 --space jobs --output json
```

## Step 4: Read Without Consuming

Query the tuple with a template. `rd` leaves the tuple in place.

```bash
tuplespacectl tuple rd --server-url http://127.0.0.1:8080 --space jobs 'job,?id:int,?ready:bool'
```

## Step 5: Consume the Tuple

Now issue the same query with `in`. This removes the tuple after the match.

```bash
tuplespacectl tuple in --server-url http://127.0.0.1:8080 --space jobs 'job,?id:int,?ready:bool'
```

## Step 6: Verify It Is Gone

Run the read query again. It should now report `not_found` or an equivalent empty result.

```bash
tuplespacectl tuple rd --server-url http://127.0.0.1:8080 --space jobs 'job,?id:int,?ready:bool'
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `not_found` on the first `rd` | The tuple was never written to that space | Re-run `tuple out` and check the `--space` value |
| Query does not match | Field types do not align | Make sure the tuple and template have the same arity and compatible types |
| Health fails | The server is unavailable | Start `tuplespaced` or correct `--server-url` |

## See Also

- `tuplespacectl help tuplespacectl-overview`
- `tuplespacectl help tuple-dsl`
