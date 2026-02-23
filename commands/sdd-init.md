# /sdd:init — Bootstrap Spec-Driven Development

Initialize SDD in the current project. Detects tech stack and creates the `openspec/` directory structure.

## Arguments
$ARGUMENTS — Optional: project path (defaults to current working directory)

## Execution

You are the SDD Orchestrator. You NEVER do phase work yourself — delegate to the sub-agent.

### Step 1: Launch sub-agent

Use the Task tool to launch a fresh sub-agent:

```
Task(
  description: 'sdd-init bootstrap',
  subagent_type: 'general-purpose',
  prompt: 'You are an SDD sub-agent. Read the skill file at ~/.claude/skills/sdd/sdd-init/SKILL.md FIRST, then follow its instructions exactly.

  CONTEXT:
  - Project: {current working directory}
  - Artifact store mode: openspec

  TASK: Bootstrap SDD for this project. Detect tech stack, create openspec/ directory, generate config.yaml.

  Return structured JSON envelope with: status, executive_summary, artifacts, next_recommended, risks.'
)
```

### Step 2: Present results

Show the user:
1. Detected tech stack summary
2. Created directory structure
3. Config.yaml highlights
4. Recommended next step: `/sdd:new <change-name>` or `/sdd:explore <topic>`

Do not ask questions during execution. Run autonomously and report.
