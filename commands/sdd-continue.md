# /sdd-continue — Run Next SDD Phase

## Arguments
$ARGUMENTS — Optional: change name. If omitted, auto-detects from `sdd list`.

## Execution

### Step 1: Detect change + phase

```bash
sdd list                # if no name given — pick one or suggest /sdd-new
sdd status <name>       # current_phase tells what runs next
```

### Step 2: Route by current_phase

#### explore, propose, spec, design, tasks

```bash
sdd context <name> [phase]
```

```
Agent(
  description: 'sdd-{phase} for {change-name}',
  model: 'sonnet',  # sonnet for explore/tasks only; omit for propose/spec/design (Opus)
  prompt: '{context output}

  Write to: openspec/changes/{change-name}/.pending/{phase}.md
  Follow the SKILL instructions exactly.'
)
```

```bash
sdd write <name> <phase>
```

**spec + design parallel:** when `current_phase` is `spec`, run both sub-agents simultaneously, promote both.

#### apply

```bash
sdd context <name> apply
```

```
Agent(
  description: 'sdd-apply for {change-name}',
  prompt: '{context output}

  Implement next incomplete task. Build-check after each task (max 3 fix attempts).
  Mark completed [x]. Write to: openspec/changes/{change-name}/.pending/apply.md'
)
```

```bash
sdd write <name> apply
```

If tasks remain → `/sdd-continue`. All done → `/sdd-review`.

#### review

```bash
sdd context <name> review
```

```
Agent(
  description: 'sdd-review for {change-name}',
  prompt: '{context output}

  Review against specs and design.
  Write to: openspec/changes/{change-name}/.pending/review.md'
)
```

```bash
sdd write <name> review
```

PASS → `/sdd-verify`. FAIL → `/sdd-apply --fix-only`.

#### verify

```bash
sdd verify <name>
```

Passed → `sdd write <name> verify`. Suggest `/sdd-clean`.
Failed → suggest `/sdd-verify --fix`.

#### clean

```bash
sdd context <name> clean
```

```
Agent(
  description: 'sdd-clean for {change-name}',
  model: 'sonnet',
  prompt: '{context output}

  Clean up modified files. Write to: openspec/changes/{change-name}/.pending/clean.md'
)
```

```bash
sdd write <name> clean
```

Suggest `/sdd-archive`.

#### archive

```bash
sdd archive <name>
```

Show archive location. Suggest `/sdd-new` or commit.
