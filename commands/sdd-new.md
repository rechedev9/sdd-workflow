# /sdd:new — Start a New SDD Change

Creates a new change by running exploration + proposal. This is the main entry point for starting work on a feature, bugfix, or refactor.

## Arguments
$ARGUMENTS — Change name (kebab-case, required). Optionally followed by a description.
Example: `/sdd:new add-csv-export Export workout data as CSV files`

## Execution

You are the SDD Orchestrator. You manage the flow and delegate to sub-agents.

### Step 1: Validate environment

- Check that openspec/ exists. If not, suggest running `/sdd:init` first.
- Check that the change name doesn't already exist in openspec/changes/

### Step 2: Extract arguments

- First word after command: change name (kebab-case)
- Remaining text: intent description
- If no description provided, ask the user for a brief intent

### Step 3: Run sdd-explore

Launch sub-agent for exploration:

```
Task(
  description: 'sdd-explore for {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'Read ~/.claude/skills/sdd/sdd-explore/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Topic: {intent description}
  - Change name: {change-name}
  - Detail level: standard
  - Config: openspec/config.yaml

  TASK: Explore codebase for this change. Create exploration.md in openspec/changes/{change-name}/.

  Return JSON envelope.'
)
```

### Step 4: Present exploration & get approval

Show user the exploration summary. Ask: "Proceed to proposal?"

### Step 5: Run sdd-propose

Launch sub-agent for proposal:

```
Task(
  description: 'sdd-propose {change-name}',
  subagent_type: 'general-purpose',
  model: 'sonnet',
  prompt: 'Read ~/.claude/skills/sdd/sdd-propose/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change name: {change-name}
  - Exploration: openspec/changes/{change-name}/exploration.md
  - Intent: {intent description}

  TASK: Create proposal.md for this change.

  Return JSON envelope.'
)
```

### Step 6: Present proposal & get approval

Show user the proposal summary with:
1. Intent
2. Scope (in/out)
3. Approach
4. Risks
5. Rollback plan

Ask: "Approve proposal? Next step: `/sdd:continue {change-name}` to generate specs + design."
