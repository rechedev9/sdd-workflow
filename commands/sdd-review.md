# /sdd-review — Semantic Code Review

## Arguments
$ARGUMENTS — Optional: change name. Flags:
- `--strict` — Treat PREFER violations as blocking
- `--security` — Run deep security scan (OWASP Top 10)
- `--quick` — Check only REJECT/REQUIRE rules, skip PREFER

## Execution

### Step 1: Get review context

```bash
sdd context <name> review
```

### Step 2: Launch sub-agent

```
Agent(
  description: 'sdd-review for {change-name}',
  prompt: '{context from sdd context output}

  Mode: {normal|strict|quick}
  Security: {true|false}

  Review all implemented code against specs, design, and project rules.
  Write review-report.md to: openspec/changes/{change-name}/.pending/review.md

  Follow the SKILL instructions exactly.'
)
```

### Step 3: Promote + advance state

```bash
sdd write <name> review
```

### Step 4: Present results

1. Verdict: PASS / FAIL
2. Blocking issues (REJECT + REQUIRE violations)
3. Spec gaps
4. Design deviations
5. Suggestions (PREFER, non-blocking)
6. Next step: PASS → `/sdd-verify {change-name}`; FAIL → `/sdd-apply --fix-only {change-name}`
