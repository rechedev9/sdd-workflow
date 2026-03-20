# /sdd-explore — Investigate Codebase

## Arguments
$ARGUMENTS — Topic or question to explore (required). Flags:
- `--deep` — Deep analysis with more detail
- `--concise` — Shorter, focused analysis
- `--for <change-name>` — Associate with an existing change

## Execution

### Step 1: Get explore context

If `--for <change-name>`:
```bash
sdd context <change-name> explore
```

Otherwise, standalone exploration — skip `sdd context`.

### Step 2: Launch sub-agent

```
Agent(
  description: 'sdd-explore {topic}',
  model: 'sonnet',
  prompt: '{context from sdd context if available, otherwise:}

  Project: {current working directory}
  Topic: {extracted topic}
  Detail level: {concise|standard|deep}

  Explore the codebase for the given topic. Produce:
  1. Current state analysis
  2. Affected areas with file paths
  3. Approach comparison (if multiple approaches exist)
  4. Recommendation
  5. Risks

  If associated with a change (--for), write exploration to:
  File: openspec/changes/{change-name}/.pending/explore.md'
)
```

### Step 3: If --for, promote

```bash
sdd write <change-name> explore
```

### Step 4: Present results

1. Executive summary (2-3 sentences)
2. Affected areas table (file path, impact)
3. Recommendation
4. Risks
5. Suggested next step: `/sdd-new <name>`

Run autonomously and report. Do not ask questions during execution.
