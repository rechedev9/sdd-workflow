---
name: sdd-clean
description: >
  Three-pass code cleanup: dead code removal, duplication & reuse analysis, and quality & efficiency review.
  Runs after verify passes. Trigger: When user runs /sdd:clean or after sdd-verify passes.
license: MIT
metadata:
  version: "2.0"
---

# SDD Clean — Code Cleanup

You are executing the **clean** phase inline. Your responsibility is to clean up code after implementation — removing dead code, eliminating duplicates, and simplifying complex expressions. You operate **only on files related to the current change** and verify that every removal is safe before committing it.

## Activation

User runs `/sdd:clean`. Reads `tasks.md` and `verify-report.md` from disk. Aborts if verify verdict is FAIL.

## Inputs

Read from disk:

| Input | Source |
|---|---|
| `changeName` | Infer from `openspec/changes/` (the active change folder) |
| `tasks.md` | `openspec/changes/{changeName}/tasks.md` |
| `verify-report.md` | `openspec/changes/{changeName}/verify-report.md` |

---

## Execution Steps

### Step 1 — Determine Scope

1. Read `openspec/config.yaml` and `CLAUDE.md` for project conventions and tech stack. Extract the top-level `commands` block from config.yaml into `CMD_TYPECHECK`, `CMD_LINT`, `CMD_TEST` variables. If config.yaml has no `commands` block, fall back to lockfile detection (same as sdd-apply Step 1.6).
2. Read `tasks.md` to identify all files created or modified in this change.
2. Build the **cleanup scope**:
   - **Primary**: Files directly created/modified in this change.
   - **Secondary**: Files that import from primary files (one level of dependents).
   - **Excluded**: Everything else. Do NOT refactor the whole project.
3. Read `verify-report.md` to understand the current quality state. If the verify verdict is FAIL, **abort** — do not clean broken code.

### Step 2 — Three-Pass Analysis

Analyze all files in the cleanup scope using **three distinct passes**. Each pass adopts a different mental model — approach each one fresh, as if you are a different reviewer.

> **Why three passes?** A single linear review develops confirmation bias. Three passes with distinct goals catch different classes of issues. This mirrors multi-agent review patterns that empirically find more problems than single-agent approaches.

---

#### Pass 1 — Dead Code & Stale References

**Goal**: Find everything that exists but shouldn't — code, imports, comments, and documentation that are no longer serving a purpose.

**1a. Dead Code Detection**

| Check | Risk Level | Description |
|---|---|---|
| Unused imports | SAFE | Imports not referenced anywhere in the file |
| Unused local variables | SAFE | Variables declared but never read |
| Unused function parameters | CAREFUL | Parameters not used in the function body (may be required by interface) |
| Unused private functions | SAFE | Private/unexported functions not called within the file |
| Unused exported functions | CAREFUL | Exported functions — must verify no external callers |
| Unreachable code | SAFE | Code after `return`, `throw`, `break`, `continue` |
| Dead branches | CAREFUL | `if` branches with conditions that are always true/false |
| Commented-out code | SAFE | Code blocks that are commented out (not documentation comments) |

**1b. Documentation Synchronization**

For every function **modified in this change** (from tasks.md), check for stale documentation. Fix misleading comments — do NOT add docs to undocumented functions.

| Check | Detection | Fix |
|---|---|---|
| **Stale JSDoc @param** | JSDoc lists a parameter that no longer exists, or is missing one that was added | Update @param list to match signature |
| **Stale JSDoc @returns** | JSDoc return description doesn't match actual return type/behavior | Update @returns |
| **Wrong @throws / @example** | Annotations reference behavior that no longer exists | Remove or update |
| **Misleading inline comments** | Comment describes logic that was changed | Rewrite to describe current behavior |
| **Orphaned TODO/FIXME** | TODO references a task completed in this change | Remove |
| **Stale variable/function name in comment** | Comment references a renamed symbol | Update the reference |

Do NOT: add new JSDoc, add explanatory comments, rewrite comments for style, touch files outside scope.

**Indirect caller audit**: If a public function's signature changed, grep for callers across the codebase. Update stale doc references in callers within scope; note out-of-scope callers in the clean report.

---

#### Pass 2 — Duplication & Reuse

**Goal**: Find code that is repeated or that reinvents something the codebase already provides. This pass looks **outward** — beyond the change scope into the existing codebase.

**2a. Duplicate Detection (within change scope)**

Scan for duplicated logic within and across files in the primary scope:

- **Identical blocks**: 5+ lines of identical code in two or more places.
- **Near-identical blocks**: Same structure with only variable names different.
- **Repeated patterns**: Same sequence of operations (e.g., fetch-parse-validate) in multiple functions.

For each duplicate, assess:
- Can it be extracted into a shared function without premature abstraction?
- Do the duplicates share the same interface and semantics?
- Would a shared function be MORE or LESS readable?

**Rule of Three**: Only consolidate if the pattern appears 3+ times, OR if 2 occurrences are truly identical (same logic, same types).

**2b. Codebase Reuse Search (beyond change scope)**

For every new function or inline logic block created in this change, **actively search** the existing codebase for pre-existing utilities that could replace it:

1. **Search helper/util directories** — `helpers/`, `utils/`, `lib/`, `shared/`, and any project-specific convention paths.
2. **Search for similar function signatures** — functions with the same parameter types and return type.
3. **Search for similar names** — functions whose names suggest the same intent (e.g., your new `formatDate` vs existing `dateToString`).
4. **Check adjacent modules** — the same directory or parent directory as the changed files.

For each match found:
- If the existing utility does exactly what the new code does → flag as **REPLACE** (use existing).
- If it does 80% of what's needed → flag as **EXTEND** (add a parameter or overload) — but only if the extension is backward-compatible.
- If it's superficially similar but semantically different → skip.

**2c. Cross-File Helper Consolidation**

When the same helper function is defined in multiple spec/test files:
1. Check if a shared helpers directory exists (e.g., `e2e/helpers/`, `test/helpers/`).
2. If the function is identical across 2+ files → extract to shared helpers.
3. If the function differs slightly → parameterize the shared version.
4. After extraction, verify all callers still work (typecheck + tests).

---

#### Pass 3 — Quality & Efficiency

**Goal**: Find code that works but is suboptimal — unnecessarily complex, slow, or wasteful. This pass thinks about **runtime behavior**, not just static structure.

**3a. Complexity Analysis**

For each function in the file:

| Metric | Threshold | Action |
|---|---|---|
| Function length | > 50 lines | Flag for potential split |
| Nesting depth | > 3 levels | Flag for early-return refactor |
| Parameter count | > 5 params | Flag for options object pattern |
| Cyclomatic complexity | > 10 | Flag for decomposition |

**3b. Simplification Opportunities**

| Pattern | Simplification |
|---|---|
| `if (x !== null && x !== undefined)` | `if (x != null)` or optional chaining |
| `x === null \|\| x === undefined ? default : x` | `x ?? default` (nullish coalescing) |
| `if (condition) { return true; } else { return false; }` | `return condition;` |
| Nested ternaries | Extract to named variables or use early returns |
| `arr.filter(...).length > 0` | `arr.some(...)` |
| `arr.filter(...).length === 0` | `!arr.some(...)` or `arr.every(x => !...)` |
| `Object.keys(obj).forEach(...)` | `for (const key of Object.keys(obj))` or `Object.entries()` |
| Manual null checks before access | Optional chaining (`?.`) |
| `try { ... } catch (e) { throw e; }` | Remove pointless try-catch |
| Redundant type annotations (inferable) | Remove only when inference is obvious and unambiguous |

**3c. Efficiency Analysis**

| Check | What to look for | Fix |
|---|---|---|
| **Redundant computations** | Same value computed multiple times in a function/render | Extract to a variable or `useMemo` |
| **Unnecessary waits/timeouts** | Hard-coded delays, excessive timeout values (especially in tests) | Reduce to minimum needed or use event-based waits |
| **N+1 patterns** | Loop that makes an API/DB call per iteration | Batch into a single call |
| **Missed concurrency** | Independent `await` calls run sequentially | Use `Promise.all()` for independent operations |
| **Hot-path bloat** | New blocking work added to startup, per-request, or per-render paths | Defer, lazy-load, or move off hot path |
| **Unbounded data structures** | Arrays/maps that grow without limit or cleanup | Add size limits or cleanup logic |
| **Listener leaks** | Event listeners or subscriptions added without cleanup | Add cleanup in `useEffect` return / `finally` block |
| **Wasted props/state** | Component receives props it never uses, or state that duplicates derived values | Remove unused props, derive instead of store |

**Efficiency in test code**: Test helpers that use `.isVisible({ timeout: N })` as a "check if present" pattern waste N milliseconds when the element is absent. Flag timeouts > 500ms in optional-presence checks.

### Step 3 — Apply Changes Safely

Aggregate findings from all three passes and apply them with verification:

#### 3a. Risk-Based Approach

**SAFE removals** (apply directly, verify after batch):
- Unused imports
- Unused local variables
- Unreachable code
- Commented-out code blocks

**CAREFUL removals** (verify after each one):
- Unused exported functions (search for callers first)
- Unused parameters (check if interface requires them)
- Dead branches (ensure condition analysis is correct)
- Duplicate consolidation

**RISKY removals** (require extra verification):
- Public API changes (exported types, interfaces)
- Removing functions used via dynamic imports (`import()`)
- Removing functions referenced in configuration files

#### 3b. Verification After Changes

After each significant removal or group of SAFE removals:

1. Run `{CMD_TYPECHECK}` — if it fails, **REVERT** the change and note it.
2. Run `{CMD_TEST}` for affected files — if tests fail, **REVERT** and note it.
3. If both pass, the change is confirmed safe.

#### 3c. Documentation Fixes

Apply all doc fixes identified in Pass 1b (Documentation Synchronization):

1. **Batch all doc fixes per file** — apply them together as a single edit operation.
2. **Verification**: Doc fixes are behavior-neutral, so they do NOT require typecheck/test verification individually. However, they are included in the final verification (Step 5).
3. **Preserve style**: Match the existing documentation style in the file (JSDoc format, comment style, indentation). If the file uses `/** */` for JSDoc, don't switch to `//`. If it uses `@param {type} name` vs `@param name - description`, match the existing convention.
4. **Minimal changes**: Fix only what is stale. Do not rephrase correct documentation for stylistic preferences.

#### 3d. Consolidating Duplicates

When consolidating duplicate code:

1. Create the shared function with:
   - A descriptive name that explains WHAT it does, not HOW.
   - Explicit parameter types and return type.
   - The same error handling pattern as the originals.
2. Replace each occurrence with a call to the shared function.
3. Verify with typecheck and tests after EACH replacement.
4. If the shared function would need more than 2 generic parameters, do NOT consolidate — the abstraction is too complex.

### Step 4 — Broader Scope Check

After cleaning primary files, check the **secondary scope** (direct dependents):

1. Are there exports from primary files that are no longer used by any dependent?
2. Are there types/interfaces in primary files that were replaced by new ones?
3. Are there old implementations that the new code supersedes?

For any findings, apply the same risk-based approach from Step 3.

### Step 5 — Final Verification

Run the full quality suite one last time:

```
{CMD_TYPECHECK}
{CMD_LINT}
{CMD_TEST}
```

All three must pass. If any fail, identify which cleanup caused the failure and revert it.

### Step 6 — Produce Clean Report

Create a persistent artifact at `openspec/changes/{changeName}/clean-report.md` with the following sections:

```markdown
# Clean Report: {changeName}

**Date**: {YYYY-MM-DD}
**Status**: SUCCESS | ERROR

## Files Cleaned
{List of each file and the actions taken on it}

## Lines Removed
{Total count and per-file breakdown}

## Actions Taken

### Pass 1 — Dead Code & Stale References
- Unused imports removed: {count}
- Dead functions removed: {count}
- Stale docs fixed: {count}

### Pass 2 — Duplication & Reuse
- Duplicates consolidated: {count}
- Replaced with existing utility: {count}
- Helpers extracted to shared module: {count}

### Pass 3 — Quality & Efficiency
- Complexity reductions: {count}
- Efficiency improvements: {count}
- Reverted changes: {count and reasons}

## Documentation Synchronization
| File | Function | Fix Type | Description |
|---|---|---|---|
| {path} | {functionName} | stale-param / stale-return / misleading-comment / orphaned-todo | {what was fixed} |

{If no stale docs found: "All documentation in modified functions is synchronized with implementation."}

## Build Status
- Typecheck: {PASS | FAIL}
- Lint: {PASS | FAIL}
- Tests: {PASS | FAIL}
```

This report serves as the audit trail for cleanup and feeds into sdd-archive.

### Step 7 — Present Summary

Write `openspec/changes/{changeName}/clean-report.md` with cleanup details and build results.

Append one JSONL line to `openspec/changes/{changeName}/quality-timeline.jsonl` (if quality tracking enabled):
```json
{ "changeName": "...", "phase": "clean", "timestamp": "...", "agentStatus": "SUCCESS|ERROR", "completeness": null, "buildHealth": { "typecheck": "PASS|FAIL", "lint": "PASS|FAIL", "tests": "PASS|FAIL" }, "issueCount": { "critical": 0 }, "phaseSpecific": { "linesRemoved": N, "filesModified": N } }
```

Present a markdown summary to the user, then STOP:

```markdown
## SDD Clean: {change_name}

**Build after cleanup**: typecheck {PASS|FAIL}  |  lint {PASS|FAIL}  |  tests {PASS|FAIL}

### Cleanup Summary
- **Files cleaned**: {N}
- **Lines removed**: {N}  |  **Unused imports**: {N}  |  **Dead functions**: {N}
- **Duplicates consolidated**: {N}  |  **Helpers extracted**: {N}
- **Complexity reductions**: {N}  |  **Efficiency improvements**: {N}

### Files Modified
{For each file: `{path}` — {list of actions}}

{If reverted: ### ⚠ Reverted Actions ({N})\n{list — these were unsafe to remove}\n}

**Artifact**: `openspec/changes/{changeName}/clean-report.md`

{If SUCCESS: **Next step**: Run `/sdd:archive` to close the change and merge delta specs into main specs.}
{If ERROR (aborted): The cleanup was aborted — verify verdict was FAIL. Fix the failing build first, then re-run `/sdd:clean`.}
```

---

## Rules — Hard Constraints

1. **Scope is limited.** Only clean files from the current change + their direct dependents. Do NOT refactor the whole project.
2. **Verify after removal.** Every CAREFUL and RISKY removal must be followed by typecheck + tests. No exceptions.
3. **Revert on failure.** If removal breaks the build or tests, REVERT immediately. Note it in `clean-report.md`.
4. **No premature abstraction.** Three similar lines are better than a clever abstraction that nobody understands. Only consolidate when it genuinely improves readability.
5. **Rule of Three.** Do not extract a shared function unless the pattern appears 3+ times (or 2 identical occurrences).
6. **Preserve public API.** Do not remove or rename exported functions/types that might be used by consumers outside your scope.
7. **Never remove dynamically referenced code.** If a function might be called via `import()`, string-based lookup, or configuration, do NOT remove it.
8. **Abort if verify failed.** If the verify-report verdict is FAIL, do NOT run cleanup. Broken code must be fixed first.
9. **No new features.** Cleanup must not change behavior. If you spot a bug, note it — do not fix it here.
10. **Respect existing tests.** All existing tests must still pass after cleanup. If a test tests dead code, remove the test AND the dead code together.
11. **Produce a clean-report.md.** Always write `openspec/changes/{changeName}/clean-report.md` as a persistent artifact.
12. **Respect framework idioms.** Do NOT flag framework-idiomatic patterns (React hooks dependency arrays, Tailwind class ordering, Zod chain methods, etc.) as dead code or simplification candidates. Framework skill loading is handled by `sdd-design`, `sdd-apply`, and `sdd-review` only.
13. **Handoff to sdd-archive.** After cleanup, sdd-verify should ideally re-run to produce an updated verify-report.md. If sdd-verify is not re-run, include build health (typecheck + lint + tests) in the clean-report.md as a mini-verification so that sdd-archive has a trustworthy quality snapshot.

---

## What NOT to Clean

- **Configuration files** (tsconfig, eslint config, package.json) — too risky without understanding the full impact.
- **Third-party code** (node_modules, vendored libraries) — never touch.
- **Generated code** (API clients, schema types) — these are regenerated, not hand-edited.
- **Test fixtures and mocks** — they may look like dead code but serve a testing purpose.
- **Feature flags** — code behind a feature flag may look dead but is intentionally dormant.
- **Polyfills and compatibility code** — may appear unused in your environment but needed in others.

---

## Edge Cases

| Situation | Action |
|---|---|
| Unused export is part of a barrel file (index.ts) | Check all consumers of the barrel file before removing |
| Function parameter unused but required by interface | Keep it, add `_` prefix if linter allows |
| Commented-out code has a NOTE explaining why | Keep it — it serves as documentation |
| Duplicate code exists but with subtle differences | Do NOT consolidate — the differences are intentional |
| File is near 600-line limit after cleanup | Good — cleanup helped. Note the improvement |
| Cleanup reduces a file below 20 lines | Consider whether the file should be merged into a parent module |

---

## PARCER Contract

```yaml
phase: clean
preconditions:
  - verify verdict is PASS or PASS_WITH_WARNINGS
postconditions:
  - no orphaned imports or dead code in changed files
  - clean-report.md confirms typecheck PASS
  - clean-report.md confirms tests PASS
```
