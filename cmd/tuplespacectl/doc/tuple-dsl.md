---
Title: Tuple DSL
Slug: tuple-dsl
Short: Reference for the compact tuple and template query syntax used by tuplespacectl.
Topics:
- tuplespace
- tuple-dsl
Commands:
- tuple
Flags:
- tuple-spec
- template-spec
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The compact DSL exists to make tuple operations practical from the shell. It covers tuple literals for `out` and template expressions for `rd` and `in` without forcing you to create JSON files.

## Tuple Literals

Tuple literals are comma-separated fields. Bare values infer their type:

- integers become `int`
- `true` and `false` become `bool`
- everything else becomes `string`

Examples:

```bash
tuplespacectl tuple out --space jobs 'job,42,true'
tuplespacectl tuple out --space jobs '("job with spaces",42,false)'
```

Double quotes force a string literal, which matters when a token could otherwise be parsed as an integer or boolean.

## Template Fields

Template expressions mix actual fields and formal bindings. A formal binding uses the `?name:type` form.

Examples:

```bash
tuplespacectl tuple rd --space jobs 'job,?id:int,?ready:bool'
tuplespacectl tuple in --space jobs 'worker,?name:string'
```

In these queries:

- `job` is an actual string field that must match exactly
- `?id:int` captures any integer field into the `id` binding
- `?ready:bool` captures any boolean field into the `ready` binding

## Multiple Arguments

`tuple out`, `tuple rd`, and `tuple in` accept multiple positional specs. Each positional argument is treated as a separate tuple or template.

```bash
tuplespacectl tuple out --space jobs 'job,1,true' 'job,2,false'
tuplespacectl tuple rd --space jobs 'job,?id:int' 'worker,?id:int'
```

## When To Use JSON Instead

Use JSON files when:

- you want checked-in fixtures for repeatable tests
- your shell quoting would get messy
- you are generating payloads from another program

## See Also

- `tuplespacectl help tuplespacectl-overview`
- `tuplespacectl help tutorial-first-roundtrip`
