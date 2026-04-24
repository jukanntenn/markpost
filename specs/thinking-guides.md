# Thinking Guides

> Expand your thinking before coding — most bugs come from "didn't think of that."

## Pre-Modification Rule

> **Before changing ANY value, ALWAYS search first!**

```bash
grep -r "value_to_change" .
```

This single habit prevents most "forgot to update X" bugs.

---

## Code Reuse

### Before Writing New Code

1. **Search first** — `grep -r "functionName" .` and `grep -r "keyword" .`
2. Ask: does a similar function exist? Is this pattern used elsewhere?

### Common Duplication Patterns

- **Copy-paste functions** → extract to shared utilities, import where needed
- **Similar components** → extend existing with props/variants
- **Repeated constants** → single source of truth

### When to Abstract

| Abstract when | Don't abstract when |
|---------------|---------------------|
| Same code appears 3+ times | Only used once |
| Logic is complex enough to have bugs | Trivial one-liner |
| Multiple people might need it | Abstraction would be more complex than duplication |

### After Batch Modifications

1. Review: did you catch all instances?
2. Search: `grep` to find any missed
3. Consider: should this be abstracted?

---

## Cross-Layer Thinking

### Before Implementing Cross-Layer Features

1. **Map the data flow**: `Source → Transform → Store → Retrieve → Transform → Display`
   - What format is the data in at each step?
   - What could go wrong?
   - Who validates?
2. **Identify boundaries**:
   - API ↔ Service: type mismatches, missing fields
   - Service ↔ Database: format conversions, null handling
   - Backend ↔ Frontend: serialization, date formats
   - Component ↔ Component: props shape changes
3. **Define contracts** at each boundary: exact input format, output format, possible errors

### Common Cross-Layer Mistakes

- **Implicit format assumptions** → explicit conversion at boundaries
- **Scattered validation** → validate once at entry point
- **Leaky abstractions** → each layer only knows its neighbors

---

## When to Use These Guides

### Code Reuse

- [ ] You're writing similar code to something that exists
- [ ] You see the same pattern repeated 3+ times
- [ ] You're adding a new field to multiple places
- [ ] You're modifying any constant or config
- [ ] You're creating a new utility/helper function

### Cross-Layer

- [ ] Feature touches 3+ layers (API, Service, Component, Database)
- [ ] Data format changes between layers
- [ ] Multiple consumers need the same data
- [ ] You're not sure where to put some logic

---

## Commit Checklist

- [ ] Searched for existing similar code
- [ ] No copy-pasted logic that should be shared
- [ ] Constants defined in one place
- [ ] Similar patterns follow same structure
- [ ] Data flow mapped for cross-layer features
- [ ] Validation happens at entry point only
