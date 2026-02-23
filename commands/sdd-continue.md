# /sdd:continue — Run Next SDD Phase

Automatically detects which phase is next for a change and runs it. Follows the dependency graph.

## Arguments
$ARGUMENTS — Optional: change name. If omitted, detects the active change from openspec/changes/.

## Phase Dependency Graph

```
explore → propose → specs + design (parallel) → tasks → apply → review → verify → clean → archive
```

## Execution

You are the SDD Orchestrator.

### Step 1: Detect active change

- If change name provided, use it
- Otherwise, scan openspec/changes/ for non-archived changes
- If multiple active changes, list them and ask user to pick one

### Step 2: Determine next phase

Read existing artifacts in `openspec/changes/{change-name}/` to determine what's done:

| Artifact Exists | Meaning |
|---|---|
| (no artifacts at all) | Nothing started — suggest `/sdd:new` instead |
| exploration.md (but no proposal.md) | Exploration done, propose is next |
| proposal.md | Propose done, spec + design are next |
| specs/\*/spec.md | Spec done |
| design.md | Design done |
| specs/\*/spec.md AND design.md | Spec + design done, tasks are next |
| tasks.md (exists but has unchecked items) | Task generation done, apply is next |
| tasks.md (all items [x]) | All implementation done, review is next |
| review-report.md | Review done, verify is next |
| verify-report.md (PASS) | Verify done, clean is next |
| (cleaned marker) | Clean done, archive is next |

### Step 3: Run next phase

Based on what's missing, run the next sub-agent(s):

**If no artifacts at all:**
No SDD change is in progress. Suggest the user run `/sdd:new` to start a new change. Do NOT proceed.

**If exploration exists but no proposal:**
Run sdd-propose:

```
Task(
  description: 'sdd-propose for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-propose/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/exploration.md

  TASK: Generate a proposal from the exploration results.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If proposal exists but no specs/design:**
Run sdd-spec AND sdd-design in PARALLEL (both depend only on proposal):

```
Task(
  description: 'sdd-spec for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-spec/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md

  TASK: Generate specifications from the proposal.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.',
  run_in_background: true
)

Task(
  description: 'sdd-design for {change-name}',
  subagent_type: 'general-purpose',
  # No model specified — inherits Opus from orchestrator.
  # sdd-design makes architecture decisions that shape the entire implementation.
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-design/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md

  TASK: Generate the design document from the proposal.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.',
  run_in_background: true
)

# Wait for both to complete before proceeding
```

**If specs + design exist but no tasks:**
Run sdd-tasks:

```
Task(
  description: 'sdd-tasks for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-tasks/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md, openspec/changes/{change-name}/specs/*/spec.md, openspec/changes/{change-name}/design.md

  TASK: Generate the implementation task list from specs and design.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If tasks exist but not all complete:**
Run sdd-apply for the next incomplete task:

```
Task(
  description: 'sdd-apply for {change-name}',
  subagent_type: 'general-purpose',
  # No model specified — inherits Opus from orchestrator.
  # sdd-apply writes production code; quality here directly impacts the codebase.
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-apply/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md, openspec/changes/{change-name}/specs/*/spec.md, openspec/changes/{change-name}/design.md, openspec/changes/{change-name}/tasks.md

  TASK: Implement the next incomplete task from tasks.md. Mark it [x] when done.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If all tasks complete but no review:**
Run sdd-review:

```
Task(
  description: 'sdd-review for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-review/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md, openspec/changes/{change-name}/specs/*/spec.md, openspec/changes/{change-name}/design.md, openspec/changes/{change-name}/tasks.md

  TASK: Review the completed implementation against specs and design.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If review passed but no verify:**
Run sdd-verify:

```
Task(
  description: 'sdd-verify for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-verify/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md, openspec/changes/{change-name}/specs/*/spec.md, openspec/changes/{change-name}/design.md, openspec/changes/{change-name}/tasks.md, openspec/changes/{change-name}/review-report.md

  TASK: Run verification (typecheck, lint, tests) to confirm the implementation is sound.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If verify passed but not cleaned:**
Run sdd-clean:

```
Task(
  description: 'sdd-clean for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-clean/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/proposal.md, openspec/changes/{change-name}/specs/*/spec.md, openspec/changes/{change-name}/design.md, openspec/changes/{change-name}/tasks.md, openspec/changes/{change-name}/review-report.md, openspec/changes/{change-name}/verify-report.md

  TASK: Clean up temporary artifacts and intermediate files from the change.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

**If cleaned but not archived:**
Run sdd-archive:

```
Task(
  description: 'sdd-archive for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-archive/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: openspec/changes/{change-name}/ (all remaining artifacts)

  TASK: Archive the completed change for historical reference.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

### Step 4: Present results and suggest next step

Show what was completed and what comes next. Always ask for approval before proceeding to the next phase.

## Parallel Execution Note

sdd-spec and sdd-design are the ONLY phases that can run in parallel. All others are sequential.
