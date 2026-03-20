# /sdd-verify — Technical Quality Gate

Run typecheck, lint, and tests. This is a **zero-token** operation when all checks pass — runs entirely in Go. When checks fail with `--fix`, launches systematic debugging (4-phase root cause analysis).

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--fix` — Auto-fix using systematic debugging (root cause → pattern analysis → hypothesis → fix)

## Execution

You are the SDD Orchestrator.

### Step 1: Run verify

```bash
sdd verify <name>
```

This runs build/lint/test commands from config.yaml in sequence, writes `verify-report.md` to the change directory, and returns JSON with pass/fail status.

**Zero tokens consumed if all checks pass.**

### Step 2: Handle results

**If passed:**
- Show verify-report.md summary (all green)
- Advance state: write a pending verify artifact and promote it:
  ```bash
  echo "# Verify Report\n\nAll checks passed." > openspec/changes/{change-name}/.pending/verify.md
  sdd write <name> verify
  ```
- Suggest next step: `/sdd-clean {change-name}`

**If failed (no --fix):**
- Show verify-report.md with error details (command, exit code, first 30 error lines)
- Suggest: `/sdd-verify {change-name} --fix` to auto-fix with root cause analysis

**If failed (with --fix):**
- Proceed to Step 3: Systematic Debugging

### Step 3: Systematic Debugging (--fix mode)

When verify fails, DO NOT blindly fix symptoms. Follow the 4-phase debugging protocol.
Max 3 debug-verify cycles. If still failing after 3 cycles, STOP and report.

#### Phase 1: Root Cause Investigation

Read `verify-report.md` and identify:
- Which command failed (build, lint, test)
- The exact error output (first 30 lines)
- Which files are involved

Then investigate:

```
Agent(
  description: 'investigate verify failure for {change-name}',
  # Opus — root cause analysis needs deep reasoning
  prompt: 'SYSTEMATIC DEBUGGING — PHASE 1: ROOT CAUSE INVESTIGATION

  The verify gate failed. Your job is to find the ROOT CAUSE, not patch symptoms.

  FAILED COMMAND: {command name from verify-report.md}
  ERROR OUTPUT:
  {error lines from verify-report.md}

  INVESTIGATION STEPS (mandatory, in order):
  1. Read the FULL error output — every line matters
  2. Identify the FIRST error (not cascading errors downstream)
  3. Read the file(s) mentioned in the error at the exact lines cited
  4. Check git diff to see what changed recently that could cause this
  5. Look for similar working code in the codebase for comparison
  6. Read the design.md and specs to understand intended behavior

  DO NOT:
  - Guess at fixes without reading the code
  - Fix cascading errors (fix the root, others resolve)
  - Skip reading the actual source at the error line
  - Propose multiple "maybe" fixes — find THE cause

  OUTPUT FORMAT:
  Root cause: {one sentence — the exact line/logic that is wrong and WHY}
  Evidence: {file:line references proving the root cause}
  Pattern: {similar working code that shows the correct approach}
  Proposed fix: {exact change needed, with before/after}
  Risk: {what else could break if we make this change}

  After analysis, apply the fix directly. Then run the failing command to verify:
  {the exact command that failed, from verify-report.md}

  If the fix works, report success. If not, report what happened.
  Do NOT modify any openspec/ files.'
)
```

#### Phase 2: Re-verify

After the debugging agent returns:

```bash
sdd verify <name>
```

**If passed:** Show success, proceed to Step 2 (passed path).

**If still failed:** Check cycle count.
- Cycle < 3: Return to Phase 1 with the NEW error (the previous fix may have introduced or uncovered a different issue)
- Cycle = 3: STOP. Report:
  ```
  Systematic debugging exhausted (3 cycles).

  Cycle 1: {root cause found → fix applied → result}
  Cycle 2: {root cause found → fix applied → result}
  Cycle 3: {root cause found → fix applied → result}

  Remaining errors:
  {current verify-report.md errors}

  Recommendation: Manual investigation needed. The issue may be architectural
  rather than a simple code bug. Consider reviewing design.md for misalignment.
  ```

## Key Principles

1. **Root cause FIRST** — never patch symptoms
2. **Evidence-based** — every fix cites file:line and explains WHY
3. **Pattern comparison** — find working code that does something similar
4. **One fix at a time** — fix root cause, re-verify, repeat if needed
5. **3-cycle limit** — if systematic debugging can't fix it, it's likely architectural
6. **Zero tokens on green** — verify itself is Go-native, only debugging uses Claude
