# /sdd:clean — Dead Code Removal & Simplification

Remove dead code, consolidate duplicates, simplify complexity in files touched by the change.

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--dry-run` — Report what would be removed without making changes
- `--aggressive` — Include exports used only once and over-abstracted utilities
- `--scope <path>` — Limit cleanup to specific directory

## Execution

You are the SDD Orchestrator.

### Step 1: Validate

- verify-report.md must exist and show PASS or PASS WITH WARNINGS
- If verify-report shows FAIL, refuse and suggest `/sdd:verify --fix` first

### Step 2: Launch sdd-clean sub-agent

```
Task(
  description: 'sdd-clean for {change-name}',
  subagent_type: 'general-purpose',
  prompt: 'Read ~/.claude/skills/sdd/sdd-clean/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change: {change-name}
  - Tasks: openspec/changes/{change-name}/tasks.md
  - Verify: openspec/changes/{change-name}/verify-report.md
  - Mode: {normal|dry-run|aggressive}
  - Scope: {path or "change-only"}

  TASK: Clean up code in files modified by this change. Remove dead code, consolidate duplicates, simplify.

  Return JSON envelope with: status, files_cleaned, lines_removed, build_status.'
)
```

### Step 3: Present results

1. Files cleaned (list with actions taken)
2. Lines removed
3. Complexity improvements (nesting depth, function length)
4. Build status after cleanup (typecheck, lint, tests)
5. Next step: `/sdd:archive {change-name}`

### Dry-Run Mode

When `--dry-run`:
- List everything that WOULD be removed/changed
- Categorize by risk: SAFE / CAREFUL / RISKY
- Do not modify any files
- Ask user to approve before running without --dry-run
