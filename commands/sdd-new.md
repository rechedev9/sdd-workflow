# /sdd-new — Start a New SDD Change

## Arguments
$ARGUMENTS — Change name (kebab-case, required), followed by description.

## Execution

### Step 1: Parse arguments

- First word: change name (kebab-case)
- Remaining text: intent description
- If no description, ask user for a brief intent

### Step 2: Create change + get explore context

```bash
sdd new <name> "<description>"
```

### Step 3: Run exploration

```
Agent(
  description: 'sdd-explore for {change-name}',
  model: 'sonnet',
  prompt: '{explore context from sdd new output}

  Write exploration to: openspec/changes/{change-name}/.pending/explore.md
  Follow the SKILL instructions exactly.'
)
```

### Step 4: Promote + present

```bash
sdd write <name> explore
```

Show exploration summary. Ask: "Proceed to proposal?"

### Step 5: Get propose context + run proposal

```bash
sdd context <name> propose
```

```
Agent(
  description: 'sdd-propose for {change-name}',
  prompt: '{propose context from sdd context output}

  Write proposal to: openspec/changes/{change-name}/.pending/propose.md
  Follow the SKILL instructions exactly.'
)
```

### Step 6: Promote + present

```bash
sdd write <name> propose
```

Show: Intent, Scope, Approach, Risks, Rollback plan.

Ask: "Approve proposal? Next step: `/sdd-continue {change-name}`"
