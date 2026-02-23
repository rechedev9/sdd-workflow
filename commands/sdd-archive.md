# /sdd:archive — Close Completed Change

Merge delta specs into main specs, archive the change folder, and capture learnings.

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--no-learn` — Skip learning capture
- `--force` — Archive even with warnings (never forces past CRITICAL issues)

## Execution

You are the SDD Orchestrator.

### Step 1: Safety check

- verify-report.md MUST exist
- If verdict has CRITICAL issues → ABORT (even with --force)
- If verdict has WARNINGS and no --force → ask user to confirm

### Step 2: Launch sdd-archive sub-agent

```
Task(
  description: 'sdd-archive {change-name}',
  subagent_type: 'general-purpose',
  prompt: 'Read ~/.claude/skills/sdd/sdd-archive/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change: {change-name}
  - Verify: openspec/changes/{change-name}/verify-report.md
  - Review: openspec/changes/{change-name}/review-report.md (if exists)
  - Specs: openspec/changes/{change-name}/specs/
  - Main specs: openspec/specs/
  - Skip learn: {true|false}

  TASK: Merge delta specs, archive change, capture learnings.

  Return JSON envelope with: status, specs_merged, archive_location, learnings.'
)
```

### Step 3: Present results

1. **Specs merged**: Which domains were updated in openspec/specs/
2. **Archive location**: openspec/changes/archive/YYYY-MM-DD-{change-name}/
3. **Learnings captured**: Patterns saved (if any)
4. **Change summary**: What was done, key decisions, duration

### Step 4: Suggest next actions

- `/commit-push-pr` — Commit and create PR for this change
- `/sdd:new <name>` — Start the next change
- If Engram is available: Remind to run `mem_session_summary` if ending the session

## Archive Contents

The archived folder contains the complete audit trail:
```
archive/YYYY-MM-DD-{change-name}/
├── exploration.md      (initial investigation)
├── proposal.md         (approved proposal)
├── specs/              (delta specifications)
├── design.md           (technical design)
├── tasks.md            (implementation checklist, all [x])
├── review-report.md    (semantic review results)
└── verify-report.md    (technical quality gate results)
```
