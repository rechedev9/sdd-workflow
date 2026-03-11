---
name: sdd-verify
description: >
  Technical quality gate. Runs typecheck, lint, tests, security audit. Compares implementation completeness against tasks/specs.
  Trigger: When user runs /sdd:verify or after sdd-review passes.
license: MIT
metadata:
  version: "1.0"
---

# SDD Verify — Technical Quality Gate

You are executing the **verify** phase inline. Your responsibility is to run **all technical quality checks** and produce a definitive pass/fail verdict. You check build health, test coverage, static analysis, security, and completeness against the task/spec plan. You **never fix issues** — you only report them with enough detail for a follow-up `/sdd:apply` fix pass or the developer to act.

## Activation

User runs `/sdd:verify [--fuzz]`. Reads `tasks.md`, spec files, `design.md`, and optionally `review-report.md` from disk.

## Inputs

Read from disk:

| Input | Source |
|---|---|
| `changeName` | Infer from `openspec/changes/` (the active change folder) |
| `tasks.md` | `openspec/changes/{changeName}/tasks.md` |
| `specs/` | `openspec/changes/{changeName}/specs/` |
| `design.md` | `openspec/changes/{changeName}/design.md` |
| `review-report.md` | `openspec/changes/{changeName}/review-report.md` (optional, if review ran) |
| `--fuzz` flag | Passed via CLI when user runs `/sdd:verify --fuzz` |

---

## Execution Steps

### Step 0 — Token Budget (MANDATORY before anything else)

- quality-timeline.jsonl: Bash("tail -n 5 openspec/changes/{changeName}/quality-timeline.jsonl"). NEVER use the Read tool on this file — Read loads the full file into context.
- Large source files (>150 lines): use `offset`/`limit` to read only the relevant section.
- Do NOT re-read a file already in context.
- Framework SKILL.md: load ONLY for frameworks present in the changed files, not the full tech stack.

### Step 0b — Load Build Commands

Read `openspec/config.yaml` and extract the top-level `commands` block. Store as variables for all subsequent steps:

- `CMD_TYPECHECK` ← `commands.typecheck` (e.g., `bun run typecheck`)
- `CMD_LINT` ← `commands.lint` (e.g., `bun run lint`)
- `CMD_TEST` ← `commands.test` (e.g., `bun test`)
- `CMD_FORMAT_CHECK` ← `commands.format_check` (e.g., `bun run format:check`)

**Fallback**: If `config.yaml` does not exist or has no `commands` block, detect the toolchain:

| File present | Package manager / language | Format check command |
|---|---|---|
| `bun.lockb` | bun | `bun run format:check` |
| `pnpm-lock.yaml` | pnpm | `pnpm run format:check` |
| `yarn.lock` | yarn | `yarn format:check` |
| `package-lock.json` | npm | `npm run format:check` |
| `go.mod` | Go | `gofmt -l .` (non-zero exit or output = unformatted files) |

For Go projects, also set:
- `CMD_TYPECHECK` ← `go build ./...`
- `CMD_LINT` ← `go vet ./...` (or `staticcheck ./...` if available)
- `CMD_TEST` ← `go test ./...`
- `CMD_FORMAT_CHECK` ← `gofmt -l .`

This ensures backward compatibility with pre-1.1 projects and multi-language monorepos.

### Step 1 — Completeness Check

1. Read `tasks.md`. Count:
   - Total tasks (all `[ ]` and `[x]` entries).
   - Completed tasks (`[x]` entries).
   - Incomplete tasks (`[ ]` entries).
2. Read all spec files in `specs/`. Count:
   - Total GIVEN/WHEN/THEN scenarios.
   - For each scenario, search the codebase for a corresponding test (look for `describe`/`it` blocks that reference the scenario keywords).
   - Count scenarios with tests vs. scenarios without tests.
3. Read `design.md`. Extract all interface definitions. For each interface, check if it exists in the codebase (search for `interface` or `type` declarations matching the name).
4. If `review-report.md` exists, read it and check for REJECT violations. If any REJECT violations exist, the verdict is automatically **FAIL** — but continue running all checks for the full report.

### Step 1b — Eval-Driven Assessment

Read the `## Eval Definitions` sections from all spec files in `specsDir/`. If no eval definitions exist (specs written before EDD), skip this step and proceed normally.

For each eval definition row:

| Eval Type | How to check | Failure maps to |
|-----------|-------------|-----------------|
| `code-based` | Search changed files and test files for a `describe`/`it` block whose name contains keywords from the scenario title | Missing = CRITICAL (critical) or WARNING (standard) |
| `model-based` | Semantically assess whether the implementation plausibly satisfies the THEN clause | Not satisfied = WARNING regardless of criticality |
| `human-based` | Note as requiring manual verification — do NOT fail the verdict | Note only |

**Threshold translation for single-run verify:**
- `pass^3 = 1.00` (critical) → test MUST exist → absence = **FAIL**
- `pass@3 ≥ 0.90` (standard) → test SHOULD exist → absence = **PASS_WITH_WARNINGS**

Compute:
- `evalsCriticalTotal` / `evalsCriticalPassing` — for pass^k score
- `evalsStandardTotal` / `evalsStandardPassing` — for pass@k score
- `evalPassRate` = (criticalPassing + standardPassing) / (criticalTotal + standardTotal)

### Step 2 — TypeScript Type Check

Run the TypeScript compiler:

```
{CMD_TYPECHECK}
```

- Capture the **full output** (stdout and stderr).
- Count the number of errors.
- For each error, extract: file path, line number, error code, and message.
- Classify: PASS (0 errors) or FAIL (1+ errors).

### Step 3 — ESLint

Run the linter:

```
{CMD_LINT}
```

- Capture the **full output**.
- Count errors and warnings separately.
- For each error, extract: file path, line number, rule name, and message.
- Classify: PASS (0 errors, warnings OK) or FAIL (1+ errors).

### Step 4 — Formatting

Run the format checker:

```
{CMD_FORMAT_CHECK}
```

- Capture the output.
- Classify: PASS (no formatting issues) or FAIL (files need formatting).
- List files that need formatting.

### Step 5 — Tests

Run the test suite:

```
{CMD_TEST}
```

- Capture the **full output**.
- Extract: total tests, passed, failed, skipped.
- For each failure, extract: test name, file path, error message, and stack trace.
- Classify: PASS (0 failures) or FAIL (1+ failures).

### Step 5b — Fault Localization Protocol (when tests fail)

When Step 5 detects one or more test failures, you MUST diagnose each failure using this structured protocol before continuing to Step 6. If all tests pass, skip this step.

#### PREMISES (Test Semantics)

For each failing test, describe step by step:

1. **Test identifier**: `describe > it` path and file location.
2. **Setup (Arrange)**: What preconditions does the test establish? List each variable, mock, or fixture with its value.
3. **Action (Act)**: What function or operation does the test invoke? Include the exact call signature.
4. **Assertion (Assert)**: What is the expected outcome? Quote the exact assertion (e.g., `expect(result.ok).toBe(true)`).

#### DIVERGENCE CLAIMS

For each failing test, generate one or more divergence claims:

- **CLAIM**: A formal statement cross-referencing a test premise with a specific code location.
  Format: "Test expects [expected behavior] (test file:line), but implementation at [source file:line] produces [actual behavior] because [root cause]."
- **EVIDENCE**: The specific line(s) of source code that cause the divergence.
- **CONFIDENCE**: `HIGH` | `MEDIUM` | `LOW` — based on whether the root cause is certain or hypothetical.

Example:

```
CLAIM: Test expects `Result.ok` to be `true` for valid credentials
  (auth.test.ts:25), but `verifyPassword()` at auth.service.ts:48 returns
  `false` because it compares raw password against hash without using
  `bcrypt.compare()`.
EVIDENCE: auth.service.ts:48 — `return password === storedHash`
CONFIDENCE: HIGH
```

Include all divergence claims in the verify report under a "Fault Localization" section. This structured diagnosis enables `sdd-apply` to fix issues precisely without guessing.

### Step 6 — Static Analysis

Scan all files created/modified in this change for prohibited patterns:

| Pattern | Severity | Description |
|---|---|---|
| `any` (as a type) | CRITICAL | Banned in production code |
| `as ` followed by a type name | CRITICAL | Type assertions banned (except `as const`) |
| `@ts-ignore` | CRITICAL | Compiler suppression banned |
| `@ts-expect-error` | CRITICAL | Compiler suppression banned |
| Non-null assertion `!.` or `!]` | CRITICAL | Non-null assertions banned |
| `console.log` | WARNING | Use structured logger instead |
| `console.error` | WARNING | Use structured logger instead |
| `console.warn` | WARNING | Use structured logger instead |
| `TODO` / `FIXME` | WARNING | Unresolved work items |
| `eval(` | CRITICAL | Security: code injection risk |
| `new Function(` | CRITICAL | Security: code injection risk |
| `innerHTML` | CRITICAL | Security: XSS risk |
| `.then(` chains > 2 deep | WARNING | Prefer async/await for multi-step |

**Important nuance for `as` detection**: Exclude `as const` (allowed), and exclude occurrences inside `.test.ts` files (assertions allowed in tests). Also exclude string literals that happen to contain "as " and import assertions (`import ... as`).

### Step 7 — Security Scan

Scan for hardcoded secrets and dangerous patterns:

| Pattern | Severity | Description |
|---|---|---|
| API key patterns (`[A-Za-z0-9_]{20,}` near `key`, `token`, `secret`) | CRITICAL | Possible hardcoded secret |
| `password = "..."` or `password: "..."` with literal string | CRITICAL | Hardcoded password |
| `.env` file committed | CRITICAL | Environment file should be gitignored |
| SQL string concatenation (`+ ` near `SELECT`, `INSERT`, `UPDATE`, `DELETE`) | CRITICAL | SQL injection risk |
| Unvalidated `fetch()` URL from user input | WARNING | SSRF risk |
| Missing input validation on API route handlers | WARNING | Injection risk |

### Step 7b — Dynamic Security Testing / Fuzz (Optional)

This step activates when:
- The `--fuzz` flag is passed, OR
- Step 7 found WARNING or CRITICAL security issues (auto-escalation), OR
- The change touches API handlers, auth logic, input parsers, or database operations (detected by scanning file paths and imports in the changed files)

If none of these conditions are met, skip this step entirely.

#### 7b-1. Identify Fuzz Targets

Scan the changed files to find functions that handle **external input boundaries**:

| Target Type | Detection Heuristic | Priority |
|---|---|---|
| API route handlers | Functions inside route definitions (`.get()`, `.post()`, `.put()`, `.delete()`, `.patch()`) | HIGH |
| Input parsers/validators | Functions that call `safeParse`, `parse`, `JSON.parse`, or accept `unknown`/`string` params from external sources | HIGH |
| Auth/session logic | Functions in files matching `*auth*`, `*session*`, `*login*`, `*token*` | HIGH |
| Database operations | Functions that call query builders, `.insert()`, `.update()`, `.delete()`, `.execute()` | MEDIUM |
| File/path handlers | Functions that use `fs.*`, `path.join`, or accept file path parameters | MEDIUM |

**Hard limits:** Select a maximum of **5 target functions** per change. Prioritize HIGH over MEDIUM.

#### 7b-2. Generate Fuzz Test Cases

For each target function, generate adversarial test cases across these categories:

| Category | Example Inputs | What It Tests |
|---|---|---|
| **Boundary values** | Empty string `""`, max-length string (10K chars), `0`, `-1`, `Number.MAX_SAFE_INTEGER`, `NaN`, `Infinity` | Edge case handling |
| **Injection payloads** | `'; DROP TABLE users--`, `<script>alert(1)</script>`, `../../etc/passwd`, `\`$(whoami)\`` | SQL/XSS/path traversal/command injection |
| **Type coercion** | `{}` where string expected, `[]` where object expected, `null`, `undefined`, nested objects with `__proto__` | Prototype pollution, type confusion |
| **Malformed data** | Truncated JSON `{"name":`, invalid UTF-8 bytes, strings with null bytes `\0`, extremely nested objects (100 levels) | Parser robustness |
| **Auth bypass** | Empty/expired/malformed tokens, tokens with modified claims, missing auth headers | Auth boundary integrity |

**Per function:** Generate a maximum of **10 test cases**, covering at least 3 of the 5 categories above.

#### 7b-3. Write Temporary Fuzz Test File

Write fuzz tests to `{targetDir}/{feature}.fuzz.test.ts`:

```typescript
import { describe, it, expect } from 'bun:test';
// import target function

describe('Fuzz: {functionName}', () => {
  describe('Boundary values', () => {
    it('should handle empty string input without throwing', () => {
      // Arrange: adversarial input
      // Act: call function
      // Assert: does not throw, returns Result.err or validation error
    });
  });

  describe('Injection payloads', () => {
    it('should sanitize SQL injection attempt in {paramName}', () => {
      // ...
    });
  });
});
```

**Test expectations:** Fuzz tests do NOT assert specific return values. They assert:
1. **No unhandled throws** — the function must not crash on adversarial input
2. **No raw error leakage** — error messages must not expose internals (stack traces, file paths, DB schema)
3. **Input validation triggers** — if the function uses `safeParse`/validation, adversarial input must be rejected (not silently accepted)
4. **No prototype pollution** — objects with `__proto__` keys must not modify global prototypes

#### 7b-4. Run Fuzz Tests

```
{CMD_TEST} {feature}.fuzz.test.ts
```

Capture the full output. For each failure:
- Extract the test name, input used, error message, and stack trace
- Classify the finding:

| Finding Type | Severity | Description |
|---|---|---|
| Unhandled throw on adversarial input | CRITICAL | Function crashes on malicious input — denial of service risk |
| Injection payload accepted without validation | CRITICAL | Input reaches processing layer unsanitized |
| Internal error details leaked | WARNING | Stack trace or file path in error response |
| Prototype pollution possible | CRITICAL | `__proto__` key modifies object prototype |
| No validation triggered | WARNING | Adversarial input passes through without any check |

#### 7b-5. Cleanup and Report

1. **If findings exist:** Keep the fuzz test file as evidence. Add findings to the verify report under a `## Dynamic Security Testing (Fuzz)` section. Each finding includes: target function, input used, category, severity, and observed behavior.
2. **If no findings:** Delete the fuzz test file. Note in the report: "Dynamic security testing: {N} functions tested, {M} test cases executed, no vulnerabilities found."
3. **Fixability:** Fuzz findings follow the standard fixability classification:
   - Missing input validation → `AUTO_FIXABLE` (add safeParse/validation)
   - Unhandled throw → `AUTO_FIXABLE` (wrap in try/catch + Result)
   - Injection accepted → `HUMAN_REQUIRED` (needs security review of the validation strategy)
   - Prototype pollution → `AUTO_FIXABLE` (filter `__proto__` keys)

### Step 8 — Dependency Audit (if available)

Attempt to run a dependency audit using the detected package manager (e.g., `bun pm audit`, `npm audit`, `pnpm audit`).

If the command is not available or not supported, skip this step and note it. If available, capture the output and count vulnerabilities by severity (critical, high, moderate, low).

### Step 9 — Compile Verify Report

Create `openspec/changes/{changeName}/verify-report.md`:

```markdown
# Verification Report: {changeName}

**Date**: {YYYY-MM-DD}
**Verifier**: sdd-verify (automated)
**Verdict**: PASS | PASS_WITH_WARNINGS | FAIL

## Completeness

- Tasks: {completed}/{total} completed
- Spec Scenarios: {covered}/{total} have corresponding tests
- Design Interfaces: {implemented}/{total} implemented

## Build Health

| Check | Status | Details |
|---|---|---|
| TypeScript | PASS/FAIL | {N} errors |
| ESLint | PASS/FAIL | {N} errors, {M} warnings |
| Formatting | PASS/FAIL | {N} files need formatting |
| Tests | PASS/FAIL | {passed} passed, {failed} failed, {skipped} skipped |

## Static Analysis

| Category | Count | Severity |
|---|---|---|
| Banned `any` usage | {N} | CRITICAL |
| Type assertions (`as Type`) | {N} | CRITICAL |
| Compiler suppressions | {N} | CRITICAL |
| Console usage | {N} | WARNING |
| TODO/FIXME markers | {N} | WARNING |

## Security

| Category | Count | Severity |
|---|---|---|
| Hardcoded secrets | {N} | CRITICAL |
| Injection risks | {N} | CRITICAL |
| XSS vectors | {N} | CRITICAL |
| Missing validation | {N} | WARNING |

## Dynamic Security Testing (Fuzz)

{If Step 7b was executed:}
- Functions tested: {N}
- Test cases generated: {M}
- Vulnerabilities found: {K}

| # | Target Function | File:Line | Category | Input | Severity | Observed Behavior |
|---|---|---|---|---|---|---|
| 1 | `handleLogin` | src/auth/login.ts:28 | Injection | `'; DROP TABLE--` | CRITICAL | Input reached query layer unsanitized |

{If Step 7b was skipped: "Dynamic security testing: skipped (no --fuzz flag, no security surface detected)"}

## Eval-Driven Assessment

{If eval definitions exist in specs:}

| Eval Type | Total | Passing | Score |
|-----------|-------|---------|-------|
| critical (pass^3) | {evalsCriticalTotal} | {evalsCriticalPassing} | {criticalScore} |
| standard (pass@3) | {evalsStandardTotal} | {evalsStandardPassing} | {standardScore} |
| **Overall** | {total} | {passing} | **{evalPassRate}** |

{List any missing critical evals (FAIL) and missing standard evals (WARNING)}

{If no eval definitions found: "Eval-Driven Assessment: skipped (no eval definitions in spec files — specs pre-date EDD)"}

## Fault Localization (if tests failed)

{Structured premises and divergence claims from Step 5b — omit this section if all tests passed}

## Issues Detail

| # | Severity | Category | File | Line | Description | Fixability | Fix Direction |
|---|---|---|---|---|---|---|---|
| 1 | CRITICAL | typecheck | src/foo.ts | 12 | Type 'string' not assignable to 'number' | AUTO_FIXABLE | Change parameter type or add type conversion at line 12 |

## Verdict Rationale

{Explanation of why the verdict is PASS, PASS_WITH_WARNINGS, or FAIL}
```

### Step 10 — Present Summary

Write `openspec/changes/{changeName}/verify-report.md` with full verification results.

Append one JSONL line to `openspec/changes/{changeName}/quality-timeline.jsonl` (if quality tracking enabled):
```json
{ "changeName": "...", "phase": "verify", "timestamp": "...", "agentStatus": "SUCCESS", "completeness": { "tasksCompleted": N, "tasksTotal": M, "specsCovered": N, "specsTotal": M }, "buildHealth": { "typecheck": "PASS|FAIL", "lint": "PASS|FAIL", "tests": "PASS|FAIL", "format": "PASS|FAIL" }, "issueCount": { "critical": N, "warnings": N }, "phaseSpecific": { "verdict": "PASS|PASS_WITH_WARNINGS|FAIL", "allAutoFixable": true } }
```

Present a markdown summary to the user, then STOP:

```markdown
## SDD Verify: {change_name}

**Verdict**: {✅ PASS | ⚠️ PASS_WITH_WARNINGS | ❌ FAIL}

### Build Health
| Check | Result | Details |
|-------|--------|---------|
| typecheck | {PASS/FAIL} | {N} errors |
| lint | {PASS/FAIL} | {N} errors, {N} warnings |
| tests | {PASS/FAIL} | {passed}/{total} passed, {N} failed |
| format | {PASS/FAIL} | {N} files need formatting |

### Completeness
- **Tasks**: {N}/{M} complete  |  **Specs**: {N}/{M} covered
- **Interfaces**: {N}/{M} implemented

### Static Analysis
- `any` usages: {N}  |  `@ts-ignore`: {N}  |  `console.*`: {N}  |  TODOs: {N}

{If security findings: ### ⛔ Security\n- Hardcoded secrets: {N}, Injection risks: {N}, XSS: {N}\n}
{If eval-driven: ### Eval Results\n- Critical: {N}/{M} passing  |  Standard: {N}/{M} passing\n}
{If FAIL: ### ⛔ Critical Issues\n{issue list with file:line and fixability}\n}
{If warnings: ### ⚠ Warnings\n{issue list}\n}

**Artifact**: `openspec/changes/{changeName}/verify-report.md`

{If PASS: **Next step**: Run `/sdd:clean` to remove dead code, or `/sdd:archive` to close the change.}
{If PASS_WITH_WARNINGS: **Next step**: Review warnings above. Run `/sdd:clean` or `/sdd:archive` when satisfied.}
{If FAIL and allAutoFixable: **Next step**: Run `/sdd:apply` in fix mode — all issues are auto-fixable.}
{If FAIL and has HUMAN_REQUIRED: **Next step**: Manually fix the HUMAN_REQUIRED issues above, then re-run `/sdd:verify`.}
```

---

## Rules — Hard Constraints

1. **Never fix issues.** Report only. Fixing is `sdd-apply`'s job or the developer's.
2. **Capture full command output.** Truncated output makes debugging impossible. Always capture stderr too.
3. **CRITICAL issues = FAIL verdict.** No exceptions. Even one type error means FAIL.
4. **WARNING issues = PASS_WITH_WARNINGS.** The change can proceed but issues should be addressed.
5. **SUGGESTION issues = informational.** They do not affect the verdict.
6. **Review-report REJECT violations = automatic FAIL.** If sdd-review found REJECT violations, the verify verdict is FAIL regardless of build health.
7. **Be precise.** Every issue must have a file path and line number. "There might be issues" is not acceptable.
8. **Run ALL checks.** Even if typecheck fails, still run lint, tests, and static analysis. The full picture is needed.
9. **Distinguish pre-existing issues.** If a test failure or lint error exists in a file NOT touched by this change, note it as "pre-existing" rather than attributing it to the change.
10. **Completeness matters.** A change that passes all builds but only implements 5/10 tasks is still incomplete. Note it clearly.

---

## Fixability Classification

Every issue in `verify-report.md` MUST include a `fixability` field. This determines whether a fix pass can proceed automatically or the user must intervene.

| Fixability | Criteria | Examples |
|---|---|---|
| `AUTO_FIXABLE` | Clear mechanical fix derivable from the error message and code context | Type errors (TS2xxx), lint violations, formatting issues, `any` usage, `console.log`, `as Type` assertions, TODO markers, missing accessibility attributes |
| `HUMAN_REQUIRED` | Requires architectural judgment, business decision, or design rethink | Missing feature logic (entire spec scenario unimplemented), security vulnerabilities needing risk assessment, pre-existing failures in untouched files blocking the build, missing test infrastructure |

**Classification rules:**
1. When in doubt, classify as `HUMAN_REQUIRED`.
2. Include a `fixDirection` field for `AUTO_FIXABLE` issues: a 1-sentence instruction for `sdd-apply`.
3. Build health failures (typecheck, lint, format) are almost always `AUTO_FIXABLE`. Test failures depend on root cause — a wrong assertion is `AUTO_FIXABLE`, a missing feature is `HUMAN_REQUIRED`.
4. The summary MUST clearly state whether all issues are `AUTO_FIXABLE` or if any require `HUMAN_REQUIRED` judgment.

---

## Verdict Decision Matrix

| Condition | Verdict |
|---|---|
| All checks PASS, all tasks complete, no CRITICAL issues | PASS |
| All checks PASS but has WARNING-level issues (TODO, console.log) | PASS_WITH_WARNINGS |
| Any CRITICAL issue (type error, test failure, security vuln, REJECT violation) | FAIL |
| Tasks incomplete (not all marked [x]) | FAIL |
| Spec scenarios without tests > 20% of total | PASS_WITH_WARNINGS |
| Spec scenarios without tests > 50% of total | FAIL |

---

## Edge Cases

| Situation | Action |
|---|---|
| `bun run typecheck` command not found | Note as CRITICAL — build infrastructure missing |
| Tests take > 5 minutes | Let them run up to 10 minutes, then timeout and note it |
| No test files exist at all | Flag as CRITICAL — untested code cannot pass verification |
| `review-report.md` not provided | Skip review-report checks, note that semantic review was skipped |
| Dependency audit not available | Skip and note — do not count as failure |
| Static analysis finds issues in node_modules | Ignore node_modules entirely — only scan project source |

---

## PARCER Contract

```yaml
phase: verify
preconditions:
  - review-report.md exists at openspec/changes/{changeName}/ (or review explicitly skipped)
  - implementation files exist on disk
postconditions:
  - verify-report.md written to openspec/changes/{changeName}/
  - verify-report.md contains all 4 build checks (typecheck, lint, tests, format)
  - verify-report.md verdict is PASS, PASS_WITH_WARNINGS, or FAIL
```
