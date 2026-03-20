# /sdd-verify — Technical Quality Gate

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--fix` — Auto-fix using systematic debugging (root cause → pattern analysis → hypothesis → fix)

## Execution

### Step 1: Run verify

```bash
sdd verify <name>
```

### Step 2: Handle results

**If passed:**
```bash
echo "# Verify Report\n\nAll checks passed." > openspec/changes/{change-name}/.pending/verify.md
sdd write <name> verify
```
Show verify-report.md summary. Suggest `/sdd-clean {change-name}`.

**If failed (no --fix):**
Show verify-report.md errors (command, exit code, first 30 error lines). Suggest `/sdd-verify {change-name} --fix`.

**If failed (with --fix):**
Proceed to Step 3.

### Step 3: Systematic Debugging (--fix mode)

Max 3 debug-verify cycles. If still failing after 3, STOP and report.

#### Phase 1: Root Cause Investigation

```
Agent(
  description: 'investigate verify failure for {change-name}',
  prompt: 'SYSTEMATIC DEBUGGING — PHASE 1: ROOT CAUSE INVESTIGATION

  FAILED COMMAND: {command name from verify-report.md}
  ERROR OUTPUT:
  {error lines from verify-report.md}

  INVESTIGATION STEPS (mandatory, in order):
  1. Read the FULL error output — every line matters
  2. Identify the FIRST error (not cascading errors downstream)
  3. Read the file(s) mentioned in the error at the exact lines cited
  4. Check git diff to see what changed recently
  5. Find similar working code in the codebase for comparison
  6. Read design.md and specs for intended behavior

  DO NOT guess fixes, patch cascading errors, or propose multiple "maybe" fixes.

  OUTPUT FORMAT:
  Root cause: {one sentence — exact line/logic wrong and WHY}
  Evidence: {file:line references}
  Pattern: {similar working code}
  Proposed fix: {exact change, before/after}
  Risk: {what else could break}

  Apply the fix. Run the failing command to verify.
  Report success or what happened. Do NOT modify any openspec/ files.'
)
```

#### Phase 2: Re-verify

```bash
sdd verify <name>
```

If passed: proceed to Step 2 (passed path).

If still failed:
- Cycle < 3: return to Phase 1 with the new error
- Cycle = 3: STOP. Report:
  ```
  Systematic debugging exhausted (3 cycles).

  Cycle 1: {root cause → fix → result}
  Cycle 2: {root cause → fix → result}
  Cycle 3: {root cause → fix → result}

  Remaining errors:
  {current verify-report.md errors}

  Recommendation: Manual investigation needed. Review design.md for architectural misalignment.
  ```
