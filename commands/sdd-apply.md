# /sdd-apply — Implement Code

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--phase N` — Implement only phase N from tasks.md
- `--tdd` — Write tests first, then implementation
- `--all` — Implement all remaining phases sequentially
- `--fix-only` — Only run build-fix loop on existing code

## Execution

### Step 1: Get apply context

```bash
sdd context <name> apply
```

### Step 2: Launch sub-agent

```
Agent(
  description: 'sdd-apply for {change-name}',
  prompt: '{context from sdd context output}

  Implement the next incomplete task. Use Edit/Write tools to modify project files.
  Mode: {normal|tdd|fix-only}
  Batch: {phase N if specified, else next incomplete}

  BUILD-CHECK PROTOCOL (mandatory after EACH task):
  1. Implement the task
  2. Run build command from config.yaml (go build ./... | npx tsc --noEmit | python -m py_compile {file} | cargo check)
  3. If FAILS: read full error, fix, re-run. Max 3 attempts. If still failing, mark BLOCKED and stop.
  4. If PASSES: mark task [x] and move to next task

  After batch complete: run full suite (build + lint + tests) and report results.

  Write updated tasks.md (completed items marked [x]) to:
  File: openspec/changes/{change-name}/.pending/apply.md

  Report per task: name, files modified, build check result.
  Report at end: tasks completed N/M, blocked N, final build/lint/test status.

  Follow the SKILL instructions exactly.'
)
```

### Step 3: Promote + advance state

```bash
sdd write <name> apply
```

### Step 4: Present results

1. Tasks completed (with per-task build status)
2. Blocked tasks (with error details)
3. Final build/lint/test status
4. Next: `/sdd-apply` if tasks remain, `/sdd-review` if all done

### Step 5: If --all mode

Loop: get context -> sub-agent -> promote for each incomplete phase. Stop if any task is BLOCKED.
