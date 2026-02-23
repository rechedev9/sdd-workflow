# Advanced Usage

Extending and customizing the SDD Workflow ecosystem.

---

## Creating Custom Skills

Skills are just markdown files with structured instructions. You can create new skills for any domain, framework, or workflow pattern.

### 1. Use the skill-creator skill

```
/sdd:explore "create a skill for [your framework]"
```

Or load the skill-creator directly:

```
Read ~/.claude/skills/frameworks/skill-creator/SKILL.md
Then create a skill for [your framework/domain]
```

### 2. Skill file structure

```markdown
---
name: your-skill-name
description: >
  One-line description of what this skill does.
  Trigger: When this skill should be loaded.
license: MIT
metadata:
  author: your-name
  version: "1.0"
---

# Skill Title

## Activation

This skill activates when:
- [Trigger condition 1]
- [Trigger condition 2]

## Input Envelope

[What the orchestrator passes to this sub-agent]

## Execution Steps

### Step 1: [First step]
[Instructions]

### Step 2: [Second step]
[Instructions]

## Rules and Constraints

1. [Hard constraint 1]
2. [Hard constraint 2]

## Error Handling

- [Error case 1]: [Action]
- [Error case 2]: [Action]

## Example Usage

[Realistic input/output example]
```

### 3. Register the skill

Add it to the framework skills lazy-loading table in `CLAUDE.md`:

```markdown
| Your Domain | Trigger description | `~/.claude/skills/frameworks/your-skill/SKILL.md` |
```

### 4. Evolve from learned patterns

If you've used `/learn` to capture patterns, cluster them into a skill with `/evolve`:

```
/evolve --execute
```

This turns 3+ related learned patterns into a structured SKILL.md automatically.

---

## Creating Custom Commands

Commands are markdown files in `~/.claude/commands/`. They define user-invokable slash commands.

### Command file format

```markdown
# /your-command — Short Description

What this command does in 1-2 sentences.

## Arguments
$ARGUMENTS — Argument description
Example: `/your-command arg1 arg2`

## Execution

### Step 1: [First action]
[Instructions for Claude]

### Step 2: [Second action]
[Instructions for Claude]

## Output
[What the user sees when complete]
```

### Example: Custom deployment command

```markdown
# /deploy — Deploy to Staging

Runs verification, builds, and deploys to the staging environment.

## Arguments
$ARGUMENTS — Optional: service name (default: all)

## Execution

### Step 1: Pre-deploy verification
```bash
bun run typecheck && bun run lint && bun test
```

### Step 2: Build
```bash
bun run build
```

### Step 3: Deploy
```bash
fly deploy --config fly.staging.toml
```

### Step 4: Health check
```bash
curl -f https://staging.example.com/health
```

## Output
Report deploy URL and health check status.
```

Save to `~/.claude/commands/deploy.md`. Invoke with `/deploy` or `/deploy api`.

---

## Multi-Project Workflows

### Shared skills across projects

Skills in `~/.claude/skills/` are global — available to all projects. This is intentional. A `react-19` skill should not be duplicated per project.

Project-specific patterns that repeat across sessions belong in learned skills:

```
/learn
# → creates ~/.claude/skills/learned/project-specific-pattern.md
```

### Multiple active changes

SDD supports multiple simultaneous changes in the same project:

```
openspec/changes/
  add-csv-export/    # Change 1 in progress
  fix-session-ttl/   # Change 2 in progress
  archive/
```

Use change names explicitly with `/sdd:continue`:

```
/sdd:continue add-csv-export
/sdd:continue fix-session-ttl
```

Without a name, `/sdd:continue` prompts you to pick from active changes.

### Blocking changes

Some changes must complete before others can start. Document this in proposals:

```markdown
## Dependencies

### Internal Dependencies
- `fix-session-ttl` must be merged before starting `add-remember-me` — the Remember Me feature depends on the fixed session TTL behavior
```

---

## Integrating SDD into CI/CD

### Pre-merge verification

Use `/sdd:verify` as a CI gate. The verify-report.md provides machine-readable results:

```yaml
# Example verify-report.md verdict
Verdict: PASS
```

A simple CI script:
```bash
# .github/workflows/sdd-verify.yml
- name: Run SDD verification
  run: |
    bun run typecheck
    bun run lint
    bun test
    # SDD verify-report is written by /sdd:verify in Claude Code
    # In CI, run the underlying commands directly
```

### Enforcing spec coverage

The `phases.verify.spec_test_coverage_fail` config key controls when verify fails based on scenario coverage. For strict projects:

```yaml
phases:
  verify:
    spec_test_coverage_fail: 20  # FAIL if >20% of scenarios lack tests
```

### AGENTS.md in CI

AGENTS.md rules are enforced by `sdd-review`. For automated enforcement in CI, consider running the review agent on every PR:

```bash
# After implementing and verifying:
# /sdd:review writes review-report.md
# Check the verdict:
grep "Status: FAILED" openspec/changes/*/review-report.md && exit 1 || exit 0
```

---

## Customizing Phase Behavior

### Adjusting the task breakdown

The `phases.tasks.max_files_per_task` setting controls granularity:

```yaml
phases:
  tasks:
    max_files_per_task: 1  # One file per task (maximum precision)
    # or
    max_files_per_task: 3  # Allow related files in one task
```

### Custom phase ordering in tasks

For non-standard architectures, the default 5-phase ordering can be adjusted:

```yaml
phases:
  tasks:
    phase_order:
      - database        # DB schema first (for data-heavy projects)
      - foundation      # Types and config
      - core
      - api             # API before UI (for backend-first teams)
      - frontend
      - testing
      - cleanup
```

### Skipping phases

For trivial changes, you can skip phases that add overhead:

```
# Skip explore for small, well-understood changes:
/sdd:new fix-typo --no-explore

# Skip clean for urgent patches:
/sdd:apply fix-session-ttl
/sdd:verify fix-session-ttl
/sdd:archive fix-session-ttl  # Skip clean
```

The orchestrator respects phase skips — missing artifacts mean that phase was intentionally omitted.

---

## SDD for Non-TypeScript Projects

SDD works with any tech stack. The `sdd-apply` skill adjusts based on `config.yaml`.

### Go project

```yaml
stack:
  runtime: go
  language: go

phases:
  verify:
    commands:
      typecheck: go build ./...
      lint: golangci-lint run
      test: go test ./...
      format_check: gofmt -l .
```

### Python project

```yaml
stack:
  runtime: python
  language: python
  frameworks:
    backend: django
    testing: pytest

conventions:
  error_handling:
    pattern: throw       # Python uses exceptions

phases:
  verify:
    commands:
      typecheck: mypy .
      lint: ruff check .
      test: pytest
      format_check: black --check .
```

### Rust project

```yaml
stack:
  runtime: rust
  language: rust

phases:
  verify:
    commands:
      typecheck: cargo check
      lint: cargo clippy
      test: cargo test
      format_check: cargo fmt --check
```

---

## Extending Engram Memory

### Custom topic key families

Beyond the default families (`architecture/*`, `bug/*`, `decision/*`, `pattern/*`, `config/*`, `discovery/*`, `learning/*`), create project-specific families:

```
mem_save(
  topic_key: "domain/billing/stripe-webhook-handling",
  content: "Always validate Stripe-Signature header before processing webhook body..."
)
```

### Searching memory

Use `mem_search` to find relevant context before starting work:

```
mem_search("oauth login implementation")
mem_search("stripe billing architecture")
```

Use `mem_context` at session start to load all relevant memories at once.

### Memory cleanup

Remove stale or incorrect memories with `mem_delete`. Update evolving topics with `mem_update` (not `mem_save`, which would create a duplicate — use `mem_suggest_topic_key` first to find the existing key).

---

## Team Usage

### Shared CLAUDE.md

The primary project conventions file. Keep it in the repo:

```
.
├── CLAUDE.md       # Team conventions — versioned
├── AGENTS.md       # AI review rules — versioned
└── openspec/
    ├── config.yaml
    └── specs/      # Living specifications — versioned
```

### Versioning specs

`openspec/specs/` contains the living specifications for your system. Check them into git:
- They document WHAT the system does (not HOW — that's the code)
- They evolve as requirements change (via `/sdd:archive` which merges delta specs)
- They serve as onboarding material for new team members

### PR workflow with SDD

Typical team workflow:

```
1. /sdd:new feature-name "Intent description"
   → exploration.md, proposal.md

2. Team reviews proposal.md in PR (not code yet — just the plan)

3. /sdd:continue feature-name
   → spec.md, design.md (parallel)

4. Team reviews spec.md and design.md
   "Do these specs match what we want to build?"

5. /sdd:apply feature-name --phase 1
   → implements foundation tasks

6. /sdd:review feature-name
   → checks against specs and AGENTS.md

7. /sdd:verify feature-name
   → typecheck, lint, tests pass

8. /commit-push-pr
   → PR created with full context (proposal → spec → design → implementation)

9. /sdd:archive feature-name
   → specs merged into openspec/specs/
```

The PR body can reference `openspec/changes/{name}/proposal.md` for reviewers who want the full context.

---

## When to Skip SDD

SDD adds structure and overhead. Not every change needs all 11 phases.

| Change type | Recommended approach |
|---|---|
| Typo fix | Direct edit — no SDD |
| One-line bug fix | Direct edit — no SDD |
| Config change | Direct edit — no SDD |
| Simple feature (1-2 files) | `/sdd:ff` (fast-forward, no pauses) |
| Standard feature (3-10 files) | Full SDD with `/sdd:new` + `/sdd:continue` |
| Complex feature (10+ files) | Full SDD + consider splitting into smaller changes |
| Multi-session feature | Full SDD with Engram memory enabled |
| Security-sensitive change | Full SDD + careful AGENTS.md rules |

The guiding principle: if you can hold the entire change in your head and describe it in one sentence, skip SDD. If you need to think it through, SDD pays for itself.

---

## Navigation

- [← 07-configuration.md](./07-configuration.md) — Configuration reference
- [↑ README.md](../README.md) — Back to start
