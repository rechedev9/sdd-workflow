# /sdd-clean — Dead Code Removal & Simplification

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--dry-run` — Report what would be removed without making changes
- `--aggressive` — Include exports used only once
- `--scope <path>` — Limit cleanup to specific directory

## Execution

### Step 1: Get clean context

```bash
sdd context <name> clean
```

### Step 2: Validate

Check that verify-report.md exists and shows PASS or PASS WITH WARNINGS.
If FAIL, refuse and suggest `/sdd-verify --fix` first.

### Step 3: Launch sub-agent

```
Agent(
  description: 'sdd-clean for {change-name}',
  model: 'sonnet',
  prompt: '{context from sdd context output}

  Mode: {normal|dry-run|aggressive}
  Scope: {path or "change-only"}

  Clean up code in files modified by this change. Remove dead code, consolidate duplicates, simplify.
  Write clean-report.md to: openspec/changes/{change-name}/.pending/clean.md

  Follow the SKILL instructions exactly.'
)
```

### Step 4: Promote + advance state

```bash
sdd write <name> clean
```

### Step 5: Present results

1. Files cleaned (list with actions taken)
2. Lines removed
3. Complexity improvements
4. Build status after cleanup
5. Next step: `/sdd-archive {change-name}`

### Dry-Run Mode

List changes without applying. Categorize by risk (SAFE/CAREFUL/RISKY). Ask user to approve before running without `--dry-run`.
