# /sdd:apply — Implement Code

Write actual code following the specs and design. Works in batches (one phase at a time from tasks.md).

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--phase N` — Implement only phase N from tasks.md
- `--tdd` — Write tests FIRST, then implementation (red-green-refactor)
- `--all` — Implement all remaining phases sequentially
- `--fix-only` — Only run the build-fix loop on existing code (no new implementation)

## Execution

You are the SDD Orchestrator.

### Step 1: Validate

- openspec/changes/{change-name}/tasks.md must exist
- design.md and specs/ must exist
- Identify which phases have incomplete tasks

### Step 2: Determine batch

- If `--phase N`: implement phase N only
- If `--all`: implement all incomplete phases sequentially
- Default: implement the next incomplete phase

### Step 3: Launch sdd-apply sub-agent

For each batch:
```
Task(
  description: 'sdd-apply phase {N} for {change-name}',
  subagent_type: 'general-purpose',
  prompt: 'Read ~/.claude/skills/sdd/sdd-apply/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change: {change-name}
  - Tasks: openspec/changes/{change-name}/tasks.md
  - Design: openspec/changes/{change-name}/design.md
  - Specs: openspec/changes/{change-name}/specs/
  - Config: openspec/config.yaml
  - Batch: Phase {N} (tasks {N}.1 through {N}.X)
  - Mode: {normal|tdd}

  TASK: Implement phase {N} tasks. Follow specs for acceptance criteria, design for constraints. Mark [x] in tasks.md as you complete each task. Run build-fix loop after.

  Return JSON envelope with: status, tasks_completed, tasks_remaining, build_status, deviations.'
)
```

### Step 4: Present results

After each phase:
1. Tasks completed count
2. Build status (typecheck, lint, tests)
3. Any deviations from design
4. Next step: `/sdd:apply --phase {N+1}` or `/sdd:review` if all done

### Step 5: If --all mode

Loop through phases, presenting results after each. Stop if any phase fails.

## Build-Fix Loop

After implementing each phase, the sub-agent automatically:
1. Runs typecheck → fixes errors (max 5 attempts)
2. Runs lint → fixes errors
3. Runs tests → fixes failures
4. Reports final status

If build-fix loop fails after max attempts, report and suggest manual intervention.
