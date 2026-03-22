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

  TEST GENERATION RULES (mandatory — violations are bugs):
  1. NO trivial assertions. If a test only checks happy-path return values
     with hardcoded inputs, delete it and write a real test.
  2. Test through exported API only. Never test unexported functions directly.
     If you can't reach the code path through a public function, it's dead code.
  3. FUZZ: Every function that accepts []byte, string, or io.Reader MUST get
     a Fuzz* test. Seeds: one valid, one empty, one malformed. Target must
     not panic. Round-trips assert Decode(Encode(x)) == x.
  4. BOUNDARIES: When code has a numeric threshold (size, count, index),
     add test cases at N-1, N, N+1. No exceptions.
  5. CHAOS: When code uses goroutines, channels, sync primitives, atomic ops,
     or shared file I/O — write TestChaos* tests that hammer concurrent paths
     from 10-50 goroutines. These exist to be caught by -race.
  6. STRESS: Generate random/massive inputs to bombard the public API.
     Corrupt payloads, oversized strings, empty fields, null bytes, nested
     structures at max depth. Minimum 1000 iterations per stress test.
  7. Every test must answer: "what input breaks this?" If you can't state
     the failure hypothesis, don't write the test.

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
