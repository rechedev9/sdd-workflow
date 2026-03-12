---
name: sdd-tasks
description: >
  Break design into phased, numbered implementation checklist. Tasks are grouped by phase and small enough for one session.
  Trigger: When both sdd-spec and sdd-design are complete, or when user runs /sdd:continue.
license: MIT
metadata:
  version: "1.0"
---

# SDD Tasks

You are executing the **tasks** phase inline. You transform the technical design and specifications into an actionable checklist where each task is specific, small, and verifiable. Tasks are the bridge between planning and implementation.

## Activation

User runs `/sdd:continue` after both spec and design are complete. Reads `openspec/changes/{changeName}/design.md`, all spec files in `openspec/changes/{changeName}/specs/`, and `proposal.md` for context.

## Prerequisites

This phase has strict dependencies:
- **sdd-spec**: MUST be complete (specs define WHAT to test)
- **sdd-design**: MUST be complete (design defines WHAT to build and WHERE)
- **sdd-propose**: MUST be approved (proposal defines scope and success criteria)

If either spec or design is missing, return `status: error` with a message indicating which prerequisite is incomplete.

## Execution Steps

### Step 1: Load Project Context

1. Read `openspec/config.yaml` for:
   - Task phase rules (`phases.tasks`)
   - Phase ordering (foundation, core, integration, testing, cleanup)
   - Task format conventions
   - Verification commands (typecheck, lint, test)
2. Note project-specific constraints:
   - File length limits
   - Testing patterns (describe/it, bun:test)
   - Type strictness requirements
   - Error handling patterns (Result<T, E>)

### Step 2: Read Design Document

Read `design.md` and extract:

1. **File Changes table**: Every file that will be created, modified, or deleted.
   - Categorize each file by its role: type definition, business logic, API handler, database, UI component, test, configuration
2. **Interfaces and Contracts**: Types that must be defined before implementation.
3. **Architecture Decisions**: Constraints that affect task ordering.
4. **Testing Strategy table**: Every test that needs to be written.
5. **Migration steps**: Database or infrastructure changes.
6. **Dependencies between files**: Which files must exist before others can be written.

### Step 3: Read Spec Files

Read all spec files and extract:

1. **Requirements list**: Every REQ-{DOMAIN}-{NNN} with its priority (MUST/SHOULD/MAY).
2. **Scenarios**: Every Given/When/Then scenario (these become test cases).
3. **Acceptance criteria summary**: The verification checklist.
4. Map each requirement to the design component that implements it.
5. Map each scenario to the test file that will verify it.

### Step 4: Read Proposal (if available)

If proposal path is provided, read to extract:
1. **Success criteria**: The ultimate verification checklist.
2. **Rollback plan**: Influences cleanup tasks.
3. **Out-of-scope items**: Ensures tasks do not accidentally include excluded work.

### Step 5: Bottom-Up Phase Assignment

Assign tasks to numbered phases following a strict **bottom-up** philosophy:

> **Build robust base abstractions first. Make the primitives flawless so the high-level logic becomes trivial.**

- Phase 1 contains the lowest-level work: types, schemas, config, shared utilities — things with zero internal dependencies.
- Each subsequent phase builds on the previous: business logic consumes types, API/UI consumes business logic, tests consume implementations, cleanup finalizes.
- Create as many or as few phases as the change requires. A 2-file bugfix might need 2 phases; a full-stack feature might need 6.
- No task may reference a file created in a later phase — if it does, reorder.
- The final phase must always include running the full verification suite and confirming proposal success criteria.

### Step 6: Write Individual Tasks

Each task must follow this format:

```markdown
- [ ] {Phase}.{Number} {Action verb} — {specific file path}, {specific change description}
```

Task quality standards:

| Criterion     | Rule                                                              |
|---------------|-------------------------------------------------------------------|
| Specific      | References a single file or tightly related pair (source + test)  |
| Actionable    | Starts with a verb: Create, Add, Modify, Update, Remove, Wire    |
| Verifiable    | Has a clear "done" state (file exists, test passes, type checks)  |
| Small          | Completable in one sdd-apply batch (roughly 1 file or logical unit) |
| Ordered       | Dependencies are respected (no task references an uncreated file) |

Task action verbs and their meanings:

| Verb     | Meaning                                           |
|----------|---------------------------------------------------|
| Create   | New file from scratch                             |
| Add      | New function, type, or export to existing file    |
| Modify   | Change existing function or type signature        |
| Update   | Change configuration, imports, or wiring          |
| Remove   | Delete file, function, or deprecated code         |
| Wire     | Connect modules (imports, providers, routes)       |
| Test     | Write test cases for a specific module            |
| Verify   | Run verification commands and check results       |
| Migrate  | Run database migration or data transformation     |

#### Task Completion States

Tasks use checkbox markers to track progress:

| Marker | State | Meaning |
|--------|-------|---------|
| `[ ]` | Pending | Not started |
| `[x]` | Complete | Fully implemented and verified |
| `[~]` | Partial | Started but not finished — remaining work listed in apply-report.md |

**Partial state (`[~]`) rules:**
- Used when a task is too large for a single batch, or when context limits are reached mid-task
- The apply agent marks `[~]` in tasks.md and lists remaining work in apply-report.md
- Re-run `/sdd:apply` for the same phase to complete `[~]` tasks before advancing to the next phase
- A task marked `[~]` counts as 0.5 for completeness metrics

### Step 7: Mark Parallelizable Tasks

Within each phase, some tasks can run in parallel. Mark them:

```markdown
### Phase 2: Core Implementation

> Tasks 2.1-2.3 can run in parallel.

- [ ] 2.1 Create — /abs/path/to/auth.service.ts, implement OAuth2 token exchange logic
- [ ] 2.2 Create — /abs/path/to/oauth.repository.ts, implement OAuth2 account storage
- [ ] 2.3 Create — /abs/path/to/oauth.validator.ts, implement OAuth2 callback validation

> Task 2.4 depends on 2.1 and 2.2.

- [ ] 2.4 Modify — /abs/path/to/user.service.ts, add linkOAuthAccount method using auth.service and oauth.repository
```

### Step 8: Add Requirement Traceability

For each testing task, reference the spec requirement it verifies:

```markdown
- [ ] 4.1 Test — /abs/path/to/auth.service.test.ts, test OAuth2 token exchange (REQ-AUTH-001, REQ-AUTH-002)
- [ ] 4.2 Test — /abs/path/to/oauth.validator.test.ts, test callback validation (REQ-AUTH-003)
- [ ] 4.3 Test — /abs/path/to/auth.api.test.ts, test /api/auth/oauth/callback endpoint (REQ-AUTH-001 scenario 2)
```

### Step 9: Write tasks.md

Create `openspec/changes/{change_name}/tasks.md`:

```markdown
# Implementation Tasks: {Change Name (title case)}

**Change**: {change_name}
**Date**: {ISO 8601 timestamp}
**Status**: pending
**Depends On**: design.md, specs/

---

## Summary

- **Total Tasks**: {count}
- **Phases**: {N}
- **Estimated Files Changed**: {count from design}
- **Test Cases Planned**: {count from specs}

## Verification Commands

After each phase, run:

```bash
bun run typecheck    # Must pass with zero errors
bun run lint         # Must pass with zero warnings
bun test             # Must pass all tests
```

---

## Phase {N}: {Phase Name} ({N} tasks)

{Description of what this phase accomplishes}

> Parallelizable: {list task numbers, or "All tasks are sequential"}

- [ ] {N}.1 {Action} — {file path}, {description}
- [ ] {N}.2 {Action} — {file path}, {description}
...

**Phase {N} Checkpoint**: {What should be true after this phase completes}

---

{Repeat for each phase. The number and names of phases are determined by bottom-up analysis, not a fixed template.}

---

## Requirement Traceability Matrix

| Requirement ID      | Task(s)            | Test Task(s)       | Status  |
|---------------------|--------------------|--------------------|---------|
| REQ-{DOMAIN}-001    | 2.1, 3.1           | 4.1                | pending |
| REQ-{DOMAIN}-002    | 2.2                | 4.2                | pending |
| REQ-{DOMAIN}-003    | 2.3, 3.2           | 4.3, 4.4           | pending |

## Success Criteria Checklist

From the proposal, all must be true when tasks are complete:

- [ ] {Criterion 1 from proposal}
- [ ] {Criterion 2 from proposal}
- [ ] {Criterion 3 from proposal}
- [ ] All delta specs pass (scenarios verified by tests)
- [ ] No type errors (`bun run typecheck`)
- [ ] No lint errors (`bun run lint`)
- [ ] All tests pass (`bun test`)
```

### Step 10: Validate Task Completeness

Before returning, validate:

1. **Every file in the design's File Changes table** has at least one task.
2. **Every requirement from specs** appears in the traceability matrix.
3. **Every requirement has at least one test task** mapping to it.
4. **Phase ordering respects dependencies** (no task references a file created in a later phase).
5. **Task numbering is sequential** within each phase (1.1, 1.2, ..., 2.1, 2.2, ...).
6. **No task is too large** (modifying more than 2-3 closely related files).
7. **Phase checkpoints are specific** (not "things work" but "typecheck passes, auth.service.ts exports all required functions").
8. **Success criteria from proposal** are all included in the final checklist.
9. **Cleanup phase includes spec merging** (moving delta specs to openspec/specs/ after verification).

### Step 11: Present Summary

Present a markdown summary to the user, then STOP. Do not proceed automatically.

**On success, output:**

```markdown
## SDD Tasks: {change_name}

**Total tasks**: {N}  |  **Requirement coverage**: {coverage_percent}%

### Tasks Written
`openspec/changes/{change_name}/tasks.md`

### Tasks by Phase
| Phase | Tasks | Parallelizable |
|-------|-------|---------------|
| {1}: {Name} | {N} | {N} |
| ... | ... | ... |

### Coverage
- **Files in design**: {N}  →  **Files with tasks**: {N}  ({coverage_percent}%)
- **Requirements in specs**: {N}  →  **With tasks**: {N}  →  **With tests**: {N}

{If warnings: ### ⚠ Warnings\n- {warning}\n}

**Next step**: Review `openspec/changes/{change_name}/tasks.md`. When ready, run `/sdd:apply --phase 1` to begin implementation (start a fresh session with `/clear` first).
```

## Rules and Constraints

1. **Tasks MUST reference specific file paths** from design.md. No vague "update the auth module" tasks.
2. **Each task should be completable in one sdd-apply batch** -- roughly one file or one tightly related logical unit.
3. **Phases follow strict bottom-up ordering.** No task may reference a file created in a later phase.
4. **Use hierarchical numbering**: 1.1, 1.2, 2.1, 2.2. This enables precise references ("complete task 2.3").
5. **Tasks depend on BOTH specs AND design.** Never generate tasks before both are complete.
6. **Include testing tasks that map to spec scenarios.** Every MUST requirement needs a test task.
7. **Mark parallelizable tasks explicitly** within each phase. This enables efficient parallel implementation.
8. **Include verification checkpoints** after each phase. These are not optional -- they catch issues early.
9. **All file paths must be absolute.** Never use relative paths.
10. **Never modify source code.** Task artifacts go in `openspec/changes/{change_name}/`.
11. **The traceability matrix must be complete.** Every requirement maps to implementation tasks and test tasks.
12. **Respect the project's task-size conventions.** If a file is being created with many functions, consider splitting into multiple tasks (e.g., "Create file with type exports" then "Add validation functions to file").
13. **Testing tasks should follow project test conventions** (describe/it blocks, one assertion per test where practical, Arrange/Act/Assert pattern).
14. **The final cleanup task must always include running the full verification suite** as defined in `openspec/config.yaml`.
15. **TEST GENERATION POLICY**: Do NOT generate speculative test tasks. ONLY include test tasks if: (A) A spec scenario requires verification via a specific test file (referenced in the Testing Strategy), (B) The design's Testing Strategy table maps a requirement to a test, OR (C) The task explicitly starts with "Test — ...". Do not add test tasks for "just in case" coverage.

## Error Handling

- If `openspec/config.yaml` does not exist: return `status: error` recommending `sdd-init`.
- If `design.md` does not exist: return `status: error` with message "Design must be complete before generating tasks."
- If no spec files exist: return `status: error` with message "Specs must be complete before generating tasks."
- If design and specs are inconsistent (design mentions files not covered by specs, or vice versa): warn but proceed, noting gaps.
- All errors include the phase name (`tasks`) and a human-readable message.

## PARCER Contract

```yaml
phase: tasks
preconditions:
  - spec files exist in openspec/changes/{changeName}/specs/
  - design.md exists at openspec/changes/{changeName}/
postconditions:
  - tasks.md written with ≥1 phase containing ≥1 task
  - each task references a spec scenario or design component
  - tasks.md contains ≥1 task across ≥1 phase
```
