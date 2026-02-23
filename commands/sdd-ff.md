# /sdd:ff — Fast-Forward All Planning Phases

Runs all planning phases sequentially without stopping for approval: explore -> propose -> spec + design (parallel) -> tasks.

## Arguments
$ARGUMENTS — Change name (required). Optionally prepend intent: `/sdd:ff add-dark-mode Add dark mode toggle to settings`

## Execution

You are the SDD Orchestrator. Fast-forward mode skips intermediate approvals.

### Step 1: Validate

- openspec/ must exist
- Change name provided
- If proposal already exists, skip to next missing phase

### Step 2: Run exploration + proposal (if needed)

If no proposal.md exists:

1. Launch sdd-explore sub-agent:

```
Task(
  description: 'sdd-explore for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-explore/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {project path}
  - Change: {change-name}
  - Config: openspec/config.yaml
  - Previous artifacts: (none — this is the first phase)
  - Intent: {user-provided intent if any}

  TASK: Explore the codebase to understand context relevant to this change.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

2. After exploration completes, launch sdd-propose sub-agent with exploration results:

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
  - Intent: {user-provided intent if any}

  TASK: Generate a proposal from the exploration results.

  Return structured JSON envelope with: status, executive_summary, detailed_report (optional), artifacts, next_recommended, risks.'
)
```

3. Do NOT stop for approval — proceed to Step 3.

### Step 3: Run spec + design in parallel

Launch both simultaneously after proposal is confirmed to exist:

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

# Wait for both to complete before proceeding to Step 4
```

### Step 4: Run tasks

Launch sdd-tasks with specs + design as input:

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

### Step 5: Present complete planning summary

Show the user a consolidated view:
1. **Exploration**: Key findings + codebase context (2 lines)
2. **Proposal**: Intent + scope (2 lines)
3. **Specs**: Requirements count + scenario count
4. **Design**: Key architecture decisions
5. **Tasks**: Phase count + task count
6. **Ready for**: `/sdd:apply {change-name}`

## Important

- Fast-forward is for experienced users who trust the planning pipeline
- All artifacts are still created — nothing is skipped, just approvals
- If any phase returns status: "blocked" or "failed", STOP and report
- The user can still review all artifacts before running /sdd:apply
