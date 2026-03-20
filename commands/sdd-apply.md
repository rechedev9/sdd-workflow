# /sdd-apply — Implement Code

Write actual code following the specs and design. Works one task batch at a time from tasks.md. Each task is verified with a build check immediately after implementation — errors are caught and fixed inline, not accumulated for later.

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--phase N` — Implement only phase N from tasks.md
- `--tdd` — Write tests FIRST, then implementation
- `--all` — Implement all remaining phases sequentially
- `--fix-only` — Only run build-fix loop on existing code (no new implementation)

## Execution

You are the SDD Orchestrator.

### Step 1: Get apply context

```bash
sdd context <name> apply
```

This assembles: current incomplete task from tasks.md, completed tasks summary, design constraints, specs, and the sdd-apply SKILL.md instructions.

### Step 2: Launch sub-agent

```
Agent(
  description: 'sdd-apply for {change-name}',
  # Opus — production code quality
  prompt: '{context from sdd context output}

  Implement the next incomplete task. Use Edit/Write tools to modify project files.
  Mode: {normal|tdd|fix-only}
  Batch: {phase N if specified, else next incomplete}

  ## BUILD-CHECK PROTOCOL (mandatory after EACH task)

  After implementing each task, you MUST run the build command before moving
  to the next task. This catches errors immediately instead of accumulating them.

  For each task:
  1. Implement the task
  2. Run the build/typecheck command from config.yaml:
     - Go: `go build ./...`
     - TypeScript: `npx tsc --noEmit`
     - Python: `python -m py_compile {file}`
     - Rust: `cargo check`
  3. If build FAILS:
     - Read the error output completely
     - Fix the error (it is almost always in the code you just wrote)
     - Re-run build to confirm fix
     - Max 3 fix attempts per task. If still failing, mark task as BLOCKED
       and report the error — do NOT move to the next task
  4. If build PASSES: mark task [x] in tasks.md and move to next task

  After all tasks in the batch are complete (or one is blocked):
  - Run the full verification suite: build + lint + tests
  - Report results per command

  Write updated tasks.md (with completed items marked [x]) to:
  File: openspec/changes/{change-name}/.pending/apply.md

  ## WHAT TO REPORT

  For each task completed, report:
  - Task name
  - Files modified
  - Build check: PASS or FAIL (with fix attempts if any)

  At the end, report:
  - Tasks completed: N/M
  - Tasks blocked: N (with error details)
  - Final build status: PASS/FAIL
  - Final lint status: PASS/FAIL
  - Final test status: PASS/FAIL

  Follow the SKILL instructions exactly.'
)
```

### Step 3: Promote + advance state

```bash
sdd write <name> apply
```

### Step 4: Present results

1. Tasks completed count (with per-task build status)
2. Any blocked tasks (with error details)
3. Final build/lint/test status
4. Next step: `/sdd-apply` again if tasks remain, or `/sdd-review` if all done

### Step 5: If --all mode

Loop: get context -> sub-agent -> promote for each incomplete phase. Stop if any task is BLOCKED.

## Why build-check per task?

- **Cost:** ~0 tokens (it's a shell command the sub-agent runs)
- **Benefit:** Errors caught at the source, not 3 phases later
- **Without it:** A typo in task 2 cascades into 8 errors by task 6, and the sub-agent wastes tokens debugging the cascade instead of the typo
- **With it:** Each task is verified green before moving on. If task 3 breaks, you know it's task 3's code, not a cascade from task 1
