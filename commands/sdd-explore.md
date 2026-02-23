# /sdd:explore — Investigate Codebase

Read-only exploration of a codebase area or idea. Produces analysis with risk assessment.

## Arguments
$ARGUMENTS — Topic or question to explore (required). Optionally append `--deep` or `--concise` for detail level.

## Execution

You are the SDD Orchestrator. Delegate to sub-agent.

### Step 1: Parse arguments

- Extract topic from $ARGUMENTS
- Detect detail_level: `--deep` → deep, `--concise` → concise, default → standard
- Detect optional change name: `--for <change-name>`

### Step 2: Launch sub-agent

```
Task(
  description: 'sdd-explore {topic}',
  subagent_type: 'general-purpose',
  prompt: 'You are an SDD sub-agent. Read ~/.claude/skills/sdd/sdd-explore/SKILL.md FIRST, then follow its instructions.

  CONTEXT:
  - Project: {current working directory}
  - Topic: {extracted topic}
  - Detail level: {concise|standard|deep}
  - Change name: {if --for provided, else "none"}
  - Config: openspec/config.yaml (read if exists)

  TASK: Explore the codebase for the given topic. Return current state analysis, affected areas with file paths, approach comparison, recommendation, and risks.

  Return structured JSON envelope with: status, executive_summary, detailed_report, artifacts, next_recommended, risks.'
)
```

### Step 3: Present results

Show the user:
1. Executive summary (2-3 sentences)
2. Affected areas table (file path, impact)
3. Approach comparison (if multiple approaches)
4. Recommendation
5. Risks
6. Suggested next step: `/sdd:new <name>` to start a change based on this exploration

Do not ask questions during execution.
