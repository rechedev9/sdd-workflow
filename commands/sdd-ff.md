# /sdd-ff — Fast-Forward All Planning Phases

## Arguments
$ARGUMENTS — Change name (required). Optional intent after name.

## Execution

No approvals between phases. If any phase fails, STOP.

### 1. Create + explore

```bash
sdd new <name> "<description>"
```

```
Agent(
  description: 'sdd-explore for {change-name}',
  model: 'sonnet',
  prompt: '{explore context from sdd new output}
  Write to: openspec/changes/{change-name}/.pending/explore.md'
)
```

```bash
sdd write <name> explore
```

### 2. Propose

```bash
sdd context <name> propose
```

```
Agent(
  description: 'sdd-propose for {change-name}',
  prompt: '{propose context}
  Write to: openspec/changes/{change-name}/.pending/propose.md'
)
```

```bash
sdd write <name> propose
```

### 3. Spec + design (parallel)

```bash
sdd context <name> spec
sdd context <name> design
```

```
Agent(
  description: 'sdd-spec for {change-name}',
  run_in_background: true,
  prompt: '{spec context}
  Write to: openspec/changes/{change-name}/.pending/spec.md'
)

Agent(
  description: 'sdd-design for {change-name}',
  prompt: '{design context}
  Write to: openspec/changes/{change-name}/.pending/design.md'
)
```

```bash
sdd write <name> spec
sdd write <name> design
```

### 4. Tasks

```bash
sdd context <name> tasks
```

```
Agent(
  description: 'sdd-tasks for {change-name}',
  model: 'sonnet',
  prompt: '{tasks context}
  Write to: openspec/changes/{change-name}/.pending/tasks.md'
)
```

```bash
sdd write <name> tasks
```

### 5. Summary

Show: exploration findings, proposal scope, spec count, design decisions, task count.
Suggest `/sdd-apply {change-name}`.
