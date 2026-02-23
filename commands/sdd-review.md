# /sdd:review — Semantic Code Review

Compare implementation against specs, design, and AGENTS.md rules. Reports issues but does NOT fix them.

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--strict` — Treat PREFER violations as blocking (default: non-blocking)
- `--security` — Run deep security scan (OWASP Top 10)
- `--quick` — Check only REJECT/REQUIRE rules, skip PREFER

## Execution

You are the SDD Orchestrator.

### Step 1: Validate

- openspec/changes/{change-name}/tasks.md must have completed tasks
- specs/ and design.md must exist
- Check for AGENTS.md in project root (optional but recommended)

### Step 2: Launch sdd-review sub-agent

```
Task(
  description: 'sdd-review for {change-name}',
  subagent_type: 'general-purpose',
  prompt: 'Read ~/.claude/skills/sdd/sdd-review/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change: {change-name}
  - Specs: openspec/changes/{change-name}/specs/
  - Design: openspec/changes/{change-name}/design.md
  - Tasks: openspec/changes/{change-name}/tasks.md
  - AGENTS.md: {project root}/AGENTS.md (if exists)
  - Mode: {normal|strict|quick}
  - Security: {true|false}

  TASK: Review all implemented code against specs, design, and AGENTS.md. Create review-report.md.

  Return JSON envelope with: status, executive_summary, issues_count, blocking_count, pass_fail.'
)
```

### Step 3: Present results

1. **Verdict**: PASS / FAIL
2. **Blocking issues** (REJECT + REQUIRE violations)
3. **Spec gaps** (scenarios not satisfied)
4. **Design deviations**
5. **Suggestions** (PREFER, non-blocking)
6. Next step:
   - If PASS: `/sdd:verify {change-name}`
   - If FAIL: `/sdd:apply --fix-only {change-name}` then re-review
