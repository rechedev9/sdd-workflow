# /sdd:verify — Technical Quality Gate

Run typecheck, lint, tests, security audit. Compare implementation completeness against tasks and specs.

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--fix` — Auto-fix issues found (run build-fix loop)
- `--security` — Include dependency audit and secrets scan
- `--full` — All checks including security

## Execution

You are the SDD Orchestrator.

### Step 1: Validate

- openspec/changes/{change-name}/ must exist with tasks.md
- Preferably review-report.md exists (run /sdd:review first)

### Step 2: Launch sdd-verify sub-agent

```
Task(
  description: 'sdd-verify for {change-name}',
  subagent_type: 'general-purpose',
  prompt: 'Read ~/.claude/skills/sdd/sdd-verify/SKILL.md FIRST.

  CONTEXT:
  - Project: {cwd}
  - Change: {change-name}
  - Tasks: openspec/changes/{change-name}/tasks.md
  - Specs: openspec/changes/{change-name}/specs/
  - Design: openspec/changes/{change-name}/design.md
  - Review: openspec/changes/{change-name}/review-report.md (if exists)
  - Security: {true|false}

  TASK: Run all quality checks. Create verify-report.md.

  Return JSON envelope with: status, verdict, typecheck, lint, tests, security, completeness.'
)
```

### Step 3: Present results

```
VERIFICATION: [PASS/PASS WITH WARNINGS/FAIL]

Completeness: [X/Y tasks, A/B scenarios]
TypeScript:   [OK/X errors]
Lint:         [OK/X issues]
Format:       [OK/FAIL]
Tests:        [X passed, Y failed]
Security:     [OK/X findings]
Static:       [OK/X issues]

Next: [/sdd:clean or /sdd:apply --fix-only]
```

### Step 4: If --fix flag

When fix mode is enabled:
1. Run sdd-verify first to identify issues
2. Launch sdd-apply with --fix-only to fix identified issues
3. Re-run sdd-verify
4. Max 3 fix-verify cycles
5. Report final status
