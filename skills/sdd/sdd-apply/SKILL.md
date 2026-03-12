---
name: sdd-apply
description: >
  Implement code following specs and design. Works in batches (one phase at a time). Includes build-fix loop.
  Trigger: When user runs /sdd:apply or after sdd-tasks completes.
license: MIT
metadata:
  version: "1.0"
---

# SDD Apply — Implementation

You are executing the **apply** phase inline. Your sole responsibility is writing production code that satisfies specs and design constraints. You work in **batches** (one phase at a time) and include a **build-fix loop** after each batch to ensure the codebase compiles, passes lint, and passes tests before stopping for user review.

## Activation

User runs `/sdd:apply [--phase <N>] [--tdd] [--dry-run]`. Reads `tasks.md`, `design.md`, and all spec files from disk.

## Inputs

Read from disk:

| Input | Source |
|---|---|
| `changeName` | Infer from `openspec/changes/` (the active change folder) |
| `tasks.md` | `openspec/changes/{changeName}/tasks.md` |
| `design.md` | `openspec/changes/{changeName}/design.md` |
| `specs/` | `openspec/changes/{changeName}/specs/` |
| `phase` | Flag `--phase <N>` or prompt user which phase to implement |
| `mode` | `'implement'` (default) or `'fix'` — fix mode is triggered by failed review/verify |
| `fixList` | (fix mode only) Issues parsed from `review-report.md` or `verify-report.md` on disk |
| `sourceGate` | (fix mode only) `'review'` or `'verify'` — which report triggered this fix pass |
| `iteration` | (fix mode only) Current negotiation iteration number (1 or 2) |

---

## Modes of Operation

The `mode` input parameter determines the agent's execution path. **All other inputs, rules, and constraints apply equally to both modes** — the difference is scope and intent.

| Mode | Default? | Trigger | Scope | Reads | Writes |
|------|----------|---------|-------|-------|--------|
| `standard` | Yes | User runs `/sdd:apply --phase N` | Full task phase from `tasks.md` | `tasks.md`, `design.md`, `specs/`, `CLAUDE.md` | New/modified source files, `tasks.md` checkboxes, `apply-report.md` |
| `fix` | No | Auto-Negotiation Loop dispatches after a FAILED review/verify gate | **Only** files listed in the fix list | `review-report.md` or `verify-report.md` (based on `sourceGate`) | **Only** files explicitly cited in the error list |

**Critical distinction**: In `standard` mode, the agent is *creative* — it interprets specs, makes design decisions, and builds features. In `fix` mode, the agent is *surgical* — it reads a precise list of issues with fix directions and applies mechanical patches. This constraint prevents hallucinated "improvements" and saves tokens during negotiation loops.

---

## Execution Steps

### Fix Mode (mode: 'fix')

When `mode` is `'fix'`, this is a **targeted repair batch** triggered by a failed review or verify gate. The agent does NOT re-implement tasks — it only fixes the listed issues.

#### F1. Load Context (Fix Mode)

1. Read `openspec/config.yaml` for project settings and `capabilities.memory_enabled`.
2. Read `CLAUDE.md` at the project root for coding conventions (these apply to fix patches too).
3. **Ignore `tasks.md` entirely.** Fix mode has no concept of tasks or phases.
4. **Read the source gate report:**
   - If `sourceGate: 'review'` → Read `openspec/changes/{changeName}/review-report.md`
   - If `sourceGate: 'verify'` → Read `openspec/changes/{changeName}/verify-report.md`
5. **Cross-reference with `fixList`** — Parse `fixList` from the report's issue table (extract all `AUTO_FIXABLE` entries), but always validate against the full report. If the report contains issues not in `fixList`, ignore them — they may be `HUMAN_REQUIRED`. Your scope is **exclusively** the `fixList` entries.

#### F2. Plan Fixes

1. Parse `fixList` entries. Group by file path.
2. For each unique file, verify it exists on disk. If a file was deleted or moved since the report was generated, add it to `fixesRemaining` with `reason: "FILE_NOT_FOUND"`.
3. Order fixes within each file by line number (ascending) to avoid offset drift when applying multiple patches to the same file.
4. **Scope validation**: Cross-check every file in `fixList` against the report's issue table. If a file appears in `fixList` but NOT in the report, skip it and log a warning — the list may be stale.

#### F3. Apply Fixes

For **each file** in the grouped fix plan:

1. **Read the file** (mandatory read-before-write — same as Step 3c in standard mode).
2. **Apply each fix** using the `fixDirection` field as the primary instruction. Follow the same type safety, error handling, and code style rules as standard implementation.
3. **Scope constraint**: You may ONLY modify the files explicitly listed in `fixList`. Do not:
   - Touch files not in the list, even if you notice issues in them.
   - Add new features, refactor surrounding code, or "improve" anything beyond the fix.
   - Create new files (unless `fixDirection` explicitly requires it, e.g., "extract type to shared module").
   - Modify `tasks.md` — fix mode does not change task completion status.
4. **Judgment gate**: If a fix in the list would require architectural changes beyond the scope of a mechanical repair (e.g., the `fixDirection` says "rename variable" but the root cause is a wrong module boundary), add it to `fixesRemaining` with `reason: "REQUIRES_HUMAN_JUDGMENT"` — do not attempt it.

#### F4. Build-Fix Loop (Fix Mode)

Run the standard **Step 4 — Build-Fix Loop** (including EET protocol). This ensures your patches don't introduce regressions. The same max-attempt ceilings apply (5 Expert / 3 Ephemeral).

**Important**: If the build-fix loop surfaces NEW errors unrelated to your fixes (pre-existing issues), do NOT fix them. Note them in `fixesRemaining` with `reason: "PRE_EXISTING"`.

#### F5. Generate Fix Report

Instead of a full `apply-report.md`, append a **fix addendum** to the existing report:

Write `openspec/changes/{changeName}/fix-report-{iteration}.md`:

```markdown
# Fix Report: {changeName} — Iteration {iteration}

**Source Gate**: {review | verify}
**Date**: {YYYY-MM-DD}
**Status**: {SUCCESS | PARTIAL | ERROR}
**Fixes Applied**: {N}/{M}

## Fixes Applied

| # | File | Line | Category | Fix Applied |
|---|------|------|----------|-------------|
| 1 | {path} | {line} | {category} | {description of what was changed} |

## Fixes Remaining

| # | File | Line | Category | Reason |
|---|------|------|----------|--------|
| 1 | {path} | {line} | {category} | {REQUIRES_HUMAN_JUDGMENT | FILE_NOT_FOUND | PRE_EXISTING} |

## Build Health After Fixes

| Check | Result |
|-------|--------|
| Typecheck | {PASS/FAIL} |
| Lint | {PASS/FAIL} |
| Tests | {PASS/FAIL} |
| Format | {PASS/FAIL} |
```

#### F6. Present Fix Summary

Present a markdown summary to the user, then STOP.

Write `openspec/changes/{changeName}/fix-report-{iteration}.md` with the fix details (files patched, fixes applied, fixes remaining, build results).

**Output:**

```markdown
## SDD Apply — Fix Pass {iteration} ({sourceGate} gate)

**Build**: typecheck {PASS|FAIL}  |  lint {PASS|FAIL}  |  tests {PASS|FAIL}

### Fixes Applied ({N}/{M})
| File | Line | Category | Action |
|------|------|----------|--------|
| {file} | {line} | {category} | {fixApplied} |

{If fixesRemaining:
### ⛔ Fixes Requiring Human Judgment ({N})
| File | Line | Category | Reason |
|------|------|----------|--------|
| {file} | {line} | {category} | REQUIRES_HUMAN_JUDGMENT |
}

{If SUCCESS: **Next step**: Re-run `/sdd:review` or `/sdd:verify` to confirm the fixes resolved the issues.}
{If PARTIAL: **Next step**: Review the remaining issues above — they require human judgment to resolve.}
{If ERROR (EET): **Next step**: Build-fix loop exhausted. Review `fix-report-{iteration}.md` for persisting errors that need manual attention.}
```

### Step 1 — Load Context (Standard Mode)

> Steps 1–3 apply to `mode: 'standard'` only. For `mode: 'fix'`, see Fix Mode above.

**TOKEN BUDGET (MANDATORY):**
- quality-timeline.jsonl: Bash("tail -n 5 openspec/changes/{changeName}/quality-timeline.jsonl"). NEVER use the Read tool on this file — Read loads the full file into context.
- Read large source files (>150 lines) using `offset`/`limit` to target only the relevant section before writing.
- Do NOT re-read a file already loaded in context this session.
- Load framework SKILL.md files ONLY for frameworks directly used in the files you will modify.

1. Read `openspec/config.yaml` (if it exists) for project-wide SDD settings.
2. Read `tasks.md` — parse the full task list. Identify tasks belonging to the target **phase**.
3. Read `design.md` — extract architecture decisions, interfaces, data flow, and constraints.
4. List files in `specs/` — these contain GIVEN/WHEN/THEN acceptance criteria.
5. Read `CLAUDE.md` at the project root for project conventions (type strictness, error handling, etc.).
6. **Detect package manager and build commands** — extract from `CLAUDE.md` (look for "Build", "Test", "Check", "CI" sections) or fall back to inspecting the project root for `package.json` scripts and lockfiles (`pnpm-lock.yaml` → pnpm, `bun.lockb` → bun, `yarn.lock` → yarn, `package-lock.json` → npm). Store as:
   - `CMD_TYPECHECK` — e.g., `pnpm run typecheck:all` or `bun run typecheck`
   - `CMD_LINT` — e.g., `pnpm run check:all` or `bun run lint`
   - `CMD_LINT_FIX` — e.g., `pnpm --filter <pkg> lint:fix` or `bun run lint:fix`
   - `CMD_TEST` — e.g., `pnpm test:all` or `bun test`
   - `CMD_FORMAT_CHECK` — e.g., included in `check:all` or `bun run format:check`
   - `CMD_FORMAT_FIX` — e.g., `pnpm prettier --write` or `bun run prettier --write`
   Use these variables in Step 4 instead of hardcoded `bun` commands.

### Step 2 — Plan the Batch

1. Filter tasks.md to only tasks in the specified **phase** that are NOT already marked `[x]`.
2. Order tasks by dependency (if task B depends on task A's output, do A first).
3. For each task, identify:
   - Which spec file(s) contain the acceptance criteria.
   - Which existing source files will be modified (read them BEFORE writing).
   - Which new files need to be created.
4. If `--dry-run` flag is set: output the plan and STOP. Do not write any code.

### Step 3 — Implement Each Task

For **each task** in the batch, in dependency order:

#### 3a. Read the Spec Scenario
- Open the matching spec file from `specs/`.
- Parse every GIVEN/WHEN/THEN scenario. These are your acceptance criteria.
- If a scenario is ambiguous, note the ambiguity in `apply-report.md` — do NOT guess.

#### 3b. Read Design Constraints
- Check `design.md` for the relevant interface definitions, data types, and architectural patterns.
- If the design specifies a particular module boundary, respect it.
- If the design is WRONG (e.g., specifies an interface that cannot satisfy the spec), **note it** in `apply-report.md`. Do NOT silently deviate.

#### 3c. Read Existing Code (Structured Reading Protocol)

**Always read before writing.** For each file you are about to modify:

1. **Before reading** — State your HYPOTHESIS (expected patterns/conventions) and EVIDENCE (which spec/design/file informed it).
2. **After reading** — Note OBSERVATIONS (key patterns with `File:Line` refs), HYPOTHESIS STATUS (`CONFIRMED` | `REFUTED` | `REFINED`), and one IMPLEMENTATION IMPLICATION sentence.

Your new code MUST follow the patterns observed. Do not introduce a new style.

#### 3d. Write Code (or Tests First if TDD)

**If `--tdd` flag is set:**
1. Write the test file first (`feature.test.ts` next to `feature.ts`).
2. Use `describe` / `it` blocks (never bare `test()`).
3. Follow Arrange / Act / Assert pattern.
4. Run `bun test <file>` — confirm the test FAILS (red).
5. Write the implementation to make the test pass (green).
6. Refactor if needed (refactor).

**TEST GENERATION POLICY:**

Do NOT generate speculative tests. ONLY write or modify tests if:
- **(A)** The task explicitly starts with "Test — ..." (a dedicated test task from sdd-tasks)
- **(B)** An existing test is broken by your intentional logic changes (fix the assertion, don't delete the test)
- **(C)** Operating in explicit `--tdd` mode (see TDD Mode Details below)

In all other cases, the acceptance criteria defined in `specs/` GIVEN/WHEN/THEN scenarios are the verification contract. Do not duplicate them as agent-written tests.

**Standard mode (no --tdd):**
1. Write or modify the implementation file.
2. Follow Hard Constraints 1–16 (see Rules section below). Additional detail for nuanced rules:

**Type Safety (supplements Constraints 3, 16)**

- `as Type` is ALLOWED inside type guard functions after a runtime check (`as Record<string, unknown>`, `as unknown`). BANNED in business logic (`as User`, `as Parameters<...>[0]`). Prefer `String()`, `Number()` over `as string`.
- No non-null assertions (`!`). For required env vars, validate at startup: `if (!dbUrl) throw new Error('DATABASE_URL is required');`
- Exported functions and named declarations MUST have explicit return types. Inline callbacks in framework DSLs MAY rely on inference.
- All parameters: explicit types. Use `readonly` unless mutation is required.

**Error Handling (supplements Constraint 4)**

- `Result<T, E>` for error paths (network, DB writes, parsing). `T | null` for absence-of-data lookups.
- Real-time connections: reconnection with exponential backoff (`MAX_RECONNECT_ATTEMPTS`, `RECONNECT_DELAY_MS`).
- Event/message processing: deduplicate by ID before processing.

**Accessibility** (when implementing UI components)

1. Include `aria-label` on interactive elements, `role` on non-semantic containers, `aria-live` on dynamic content regions.
2. Use semantic HTML elements (`<time>`, `<nav>`, `<article>`, `<button>`) where appropriate.
3. These rules apply regardless of whether a framework skill is loaded.

#### 3e. Mark Task Progress
- **Fully complete**: Change `[ ]` to `[x]` in tasks.md. Add a brief note if the implementation deviated from design (e.g., `[x] Task name — NOTE: used Map instead of Record per perf requirement`).
- **Partially complete**: If a task is too large for the current batch or context limits are reached mid-task, change `[ ]` to `[~]`. Add a note describing what was completed and what remains (e.g., `[~] 2.3 Create — auth.service.ts — NOTE: token exchange implemented, refresh flow pending`). List the remaining work in `phaseSpecificData.pendingSubtasks`. Return `status: PARTIAL`.

### Step 4 — Build-Fix Loop (with Experience-Driven Early Termination)

After ALL tasks in the batch are implemented, run the build-fix loop. Before starting any fix cycle, and again before each fix attempt from attempt #3 onward, execute the EET protocol:

#### EET Protocol (Experience-Driven Early Termination)

Read `openspec/config.yaml` and check `capabilities.memory_enabled` to determine which mode to use:

**Branch A — Expert Mode (`memory_enabled: true`)**

Full EET with cross-session memory. Max **5 fix attempts** per error.

1. **Capture Error Signature**: Extract a normalized fingerprint: `{errorCode}:{affectedFile}:{errorCategory}` (e.g., `TS2345:src/auth/session.ts:type-mismatch`).
2. **Query Memory**: Run `mem_search` with the error signature as a natural language query (e.g., `"type mismatch error in auth session module"`). Semantic matching handles vocabulary variations automatically. Look for results with `bug/*` topic keys.
3. **Evaluate Trajectory**:
   - If memory returns a match where the same error persisted despite ≥3 similar fix attempts in a prior session → **EARLY TERMINATION**. Set status to `ERROR`, include the match reference in `phaseSpecificData.earlyTermination`, set `phaseSpecificData.earlyTerminationTriggered: true`, present error summary to user and STOP.
   - If no match found → proceed with fix attempt normally.
4. **Save on Escalation**: When a fix cycle exhausts all 5 attempts, save the failure pattern via `mem_save` with topic key `bug/build-fix/{errorSignature}`, content describing error details and all attempted fixes, and tags `["bug", "build-fix"]`.

**Branch B — Ephemeral Mode (`memory_enabled: false`)**

Local EET with no cross-session memory. Max **3 fix attempts** per error (more aggressive escalation since there's no historical data to guide fixes).

1. **Capture Error Signature**: Same as Branch A — extract `{errorCode}:{affectedFile}:{errorCategory}`.
2. **Track In-Session**: Maintain a local map of `{errorSignature → attemptCount}` for the current session only.
3. **Evaluate Trajectory**:
   - If the same error signature has appeared ≥3 times in this session → **EARLY TERMINATION**. Set status to `ERROR`, note `"Ephemeral Mode: error repeated 3 times without resolution"` in `phaseSpecificData.earlyTermination`, set `phaseSpecificData.earlyTerminationTriggered: true`, present error summary to user and STOP.
   - Otherwise → proceed with fix attempt.
4. **No persistence**: Do not attempt any `mem_*` calls. Failure patterns are lost at session end.

**Common to both branches**: EET is an additional **smart stop** on top of the per-attempt ceiling. When EET triggers, present the error summary to the user for manual resolution.

#### 4a. TypeScript Type Check
```
{CMD_TYPECHECK}   # e.g., pnpm run typecheck:all  OR  bun run typecheck
```
- If errors: read each error, fix the root cause (not the symptom).
- Max fix attempts per unique error: **5** (Expert Mode) or **3** (Ephemeral Mode). If still failing, flag for manual review.

#### 4b. Lint + Format
```
{CMD_LINT}   # e.g., pnpm run check:all  OR  bun run lint
```
- If errors: fix them. Prefer `{CMD_LINT_FIX}` for auto-fixable issues.
- For non-auto-fixable issues: fix manually.
- **Note**: some projects (e.g., pnpm monorepos) run Prettier as part of their lint/check command — do NOT assume a separate format step is needed if `CMD_FORMAT_CHECK` is already covered by `CMD_LINT`.
- Max fix attempts per unique error: **5** (Expert) or **3** (Ephemeral).

#### 4c. Tests
```
{CMD_TEST}   # e.g., pnpm test:all  OR  bun test
```
- If failures in files YOU touched: fix them.
- If failures in files you did NOT touch: note them in `apply-report.md` as pre-existing failures but do NOT fix them.
- Max fix attempts per unique test failure: **5** (Expert) or **3** (Ephemeral).

#### 4d. Format Check (if not covered by 4b)
```
{CMD_FORMAT_CHECK}   # e.g., pnpm prettier --check  OR  bun run format:check
```
- Only run this step if Prettier/formatting is NOT already checked as part of `CMD_LINT` (Step 4b).
- If formatting issues: run `{CMD_FORMAT_FIX} <path>` for each affected file.

### Step 5 — Generate Apply Report

Write a **change manifest** to `openspec/changes/{changeName}/apply-report.md`. This artifact creates an explicit chain of custody between apply and review — the reviewer audits exactly what was changed, not what was planned.

```markdown
# Apply Report: {changeName}

**Phase**: {phase name}
**Date**: {YYYY-MM-DD}
**Status**: {SUCCESS | PARTIAL | ERROR}
**Tasks Completed**: {N}/{M}

## Files Created

| File | Purpose |
|------|---------|
| {path} | {brief description from task} |

## Files Modified

| File | Changes |
|------|---------|
| {path} | {brief summary of modifications} |

## Files Deleted

| File | Reason |
|------|--------|
| {path} | {why it was removed} |

## Build Health

| Check | Result |
|-------|--------|
| Typecheck | {PASS/FAIL} |
| Lint | {PASS/FAIL} |
| Tests | {PASS/FAIL} ({passed}/{total}) |
| Format | {PASS/FAIL} |

## Deviations

{List any deviations from design or spec, or "None."}

## Manual Review Needed

{List any unresolved issues after build-fix loop, or "None."}
```

`apply-report.md` is the primary artifact — it feeds into sdd-review.

### Step 6 — Present Summary

Present a markdown summary to the user, then STOP. Do not proceed to the next phase automatically.

Append one JSONL line to `openspec/changes/{changeName}/quality-timeline.jsonl` (if quality tracking is enabled):
```json
{ "changeName": "...", "phase": "apply", "timestamp": "...", "agentStatus": "SUCCESS|PARTIAL|ERROR", "completeness": { "tasksCompleted": N, "tasksTotal": M }, "buildHealth": { "typecheck": "PASS|FAIL", "lint": "PASS|FAIL", "tests": "PASS|FAIL" }, "issueCount": { "critical": N }, "phaseSpecific": { "phase": "phase-N", "earlyTermination": false } }
```

**Output:**

```markdown
## SDD Apply — Phase {N} Complete

**Build**: typecheck {PASS|FAIL}  |  lint {PASS|FAIL}  |  tests {PASS|FAIL}  |  format {PASS|FAIL}

### Tasks Completed ({N}/{M})
{[x] task list from tasks.md}

{If tasksRemaining (partial batch):
### Tasks Remaining
{[ ] task list}
}

### Files Changed
- **Created**: {N} files — {list}
- **Modified**: {N} files — {list}

{If deviations:
### ⚠ Deviations from Design
- {task}: {description} → {resolution}
}

{If manualReviewNeeded:
### ⛔ Manual Review Required ({N})
{file}: {reason}
}

**Artifact**: `openspec/changes/{changeName}/apply-report.md`

{If all tasks done: **Next step**: Run `/sdd:review` to perform semantic code review.}
{If more phases remain: **Next step**: Run `/sdd:apply --phase {N+1}` to implement the next phase (start fresh with `/clear` first).}
{If EET triggered: **Next step**: Review the manual items above. When resolved, re-run `/sdd:apply --phase {N}` to complete the remaining tasks.}
```

---

## Rules — Hard Constraints

1. **Read before write.** Never modify a file you haven't read in this session.
2. **Follow existing patterns.** Your code must look like it belongs in the codebase.
3. **No type suppression.** No `any`, no `@ts-ignore`, no `!` assertions. `as Type` is allowed ONLY inside type guard functions after a runtime check — banned in business logic.
4. **Result pattern for errors.** Use `Result<T, E>` for error paths (network, DB writes, parsing). Use `T | null` for absence-of-data lookups.
5. **Note deviations.** If design is wrong, say so — don't silently deviate.
6. **Mark progress.** Update `tasks.md` as you go, not at the end.
7. **Build-fix loop is mandatory.** Never return without running typecheck + lint + tests.
8. **Max 5 attempts.** If a build error survives 5 fix attempts, escalate to manual review.
9. **One batch = one phase.** Do not implement tasks from other phases.
10. **Load domain skills.** If touching React code, load react-19 skill. If touching Tailwind, load tailwind-4 skill. Etc.
11. **Security.** Never hardcode secrets. Validate external input. No `eval()` or `innerHTML`.
12. **File size.** Aim for 200 lines per file. 600 lines hard max. Split if approaching 200.
13. **Dependency injection.** Modules accessing external resources (DB, HTTP, FS) MUST accept them as parameters — no global singletons.
14. **Bottom-up ordering.** Implement tasks in the order defined by tasks.md phases — base abstractions before consumers. Never define reusable types inline in feature modules.
15. **No `console.*`.** Use the project's structured logger. Only exception: error boundaries as last resort.
16. **Discriminated unions for state.** Multi-state objects use a discriminant field — no nullable variant-specific properties.
17. **Fix mode is scoped.** In `mode: 'fix'`, ONLY modify files in `fixList`. No new features, no refactoring, no task progress updates. Violations of this rule cause the negotiation loop to diverge.

---

## Error Recovery

| Situation | Action |
|---|---|
| Spec is ambiguous | Note ambiguity, implement most reasonable interpretation, flag in apply-report.md |
| Design contradicts spec | Follow spec (it's the source of truth), note deviation |
| Existing code has bugs | Do NOT fix unrelated bugs. Note them if they block your task |
| Test framework not set up | Create test file following project conventions, note if test infra is missing |
| Circular dependency | Refactor to break cycle using dependency injection, note in apply-report.md |
| File exceeds 600 lines | Split into focused modules, update imports |
| Environment variable missing | Validate at startup with descriptive error message, never use `!` assertion |
| Framework skill not loaded | Apply cross-framework fallback patterns (accessibility, resilience) from this file |
| Real-time connection drops | Implement reconnection with exponential backoff and configurable max attempts |
| Fix mode: `fixDirection` is vague | Apply the most conservative interpretation. If unclear, add to `fixesRemaining` with `REQUIRES_HUMAN_JUDGMENT` |
| Fix mode: fix introduces new errors | Build-fix loop handles it. If the new error is in a file NOT in `fixList`, add to `fixesRemaining` with `PRE_EXISTING` |
| Fix mode: file was refactored since report | Line numbers may be stale. Use surrounding code context to locate the issue. If unlocatable, add to `fixesRemaining` with `FILE_CHANGED` |

---

## TDD Mode Details

**When to use TDD mode**: TDD is most valuable when specs are underspecified, when the implementation involves complex algorithmic edge cases not captured in GIVEN/WHEN/THEN scenarios, or when working on pure utility functions with high combinatorial input space. If specs already provide comprehensive scenario coverage, prefer standard mode to conserve token budget. Consider spec density when deciding whether to use `--tdd`.

When `--tdd` flag is active, the implementation order inverts:

1. **Red**: Write a failing test that encodes the spec scenario.
2. **Green**: Write the minimum code to make the test pass.
3. **Refactor**: Clean up without changing behavior.
4. Repeat for each scenario in the spec.

Test file conventions:
- File: `{feature}.test.ts` alongside `{feature}.ts`
- Structure: `describe('{Feature}')` > `describe('{method/scenario}')` > `it('should ...')`
- One assertion per `it` block where practical.
- Use dependency injection over mocking. Use `mock()` from `bun:test` when DI is impractical.
- Arrange / Act / Assert — with blank lines separating each section.

---

## Cross-Framework Fallback Patterns

When the relevant framework skill IS loaded, defer to that skill over these fallbacks. When unavailable, apply these baseline patterns (complementing the rules in Step 3d):

**UI Components (React, Preact, Solid, etc.)**
- Wrap callbacks passed to child components in memoization hooks (`useCallback` / equivalent).
- Use `aria-label`, `role`, `aria-live` on interactive/dynamic elements.
- Use semantic HTML (`<time>`, `<nav>`, `<article>`, `<button>`).
- Split components at 200 lines — extract sub-components into separate files.

**Real-Time Connections (SSE, WebSocket)**
- Implement reconnection with exponential backoff (configurable `MAX_RECONNECT_ATTEMPTS`, `RECONNECT_DELAY_MS`).
- Model connection state as a discriminated union (e.g., `connecting | open | error | closed`).
- Deduplicate incoming messages by ID before processing.

**API Handlers (Elysia, Express, Hono, etc.)**
- Use `safeParse` (not `parse`) for input validation — return 400 with structured error, never throw.
- DELETE returns 204 No Content on success.
- Inline route callbacks may rely on framework type inference; extracted named handlers MUST have explicit return types.

---

## PARCER Contract

```yaml
phase: apply
preconditions:
  standard_mode:
    - tasks.md exists with ≥1 uncompleted task
    - design.md exists at openspec/changes/{changeName}/
    - spec files exist in openspec/changes/{changeName}/specs/
  fix_mode:
    - review-report.md OR verify-report.md exists at openspec/changes/{changeName}/
    - fixList is non-empty in the dispatch payload
postconditions:
  standard_mode:
    - ≥1 task marked [x] in tasks.md
    - apply-report.md written to openspec/changes/{changeName}/
    - apply-report.md contains build health (typecheck, lint, tests, format)
    - apply-report.md lists files created and modified
  fix_mode:
    - fix-report-{iteration}.md written to openspec/changes/{changeName}/
    - fix-report contains fixes applied and fixes remaining
    - fix-report contains build health (typecheck, lint, tests, format)
```
