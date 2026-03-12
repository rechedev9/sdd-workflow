---
name: sdd-init
description: >
  Bootstrap Spec-Driven Development for a project. Detects tech stack, creates openspec/ directory structure, and generates config.yaml.
  Trigger: When user runs /sdd:init or starts SDD for the first time in a project.
license: MIT
metadata:
  version: "1.0"
---

# SDD Init

You are executing the **init** phase inline. Your job is to detect the project's technology stack, architecture patterns, and conventions, then create the `openspec/` directory structure with a comprehensive `config.yaml`.

## Activation

User runs `/sdd:init`. The project root is the current working directory. Flags:
- `--force`: Overwrite existing `openspec/`
- `--dry-run`: Report what would be created without writing files

## Execution Steps

### Step 1: Check for Existing SDD Setup

1. Check if `openspec/` directory already exists at the project root.
2. If it exists and `force` is false:
   - Read `openspec/config.yaml`
   - Report the current SDD state (schema version, detected stack, number of specs, active changes)
   - Present a summary noting that SDD was already initialized, with the current config summary and next steps
3. If it exists and `force` is true:
   - Back up existing `config.yaml` as `config.yaml.bak`
   - Proceed with re-detection

### Step 2: Detect Technology Stack

Scan the project root for manifest files, lockfiles, and config files. Detect:

1. **Language, runtime, and package manager**
2. **Frameworks** (frontend, backend, ORM)
3. **Test runner, linter, and formatter**
4. **Build/check commands** (from scripts or build system)

Also read `CLAUDE.md` (conventions) if it exists.

### Step 3: Detect Architecture Patterns

Determine:

1. **Monorepo vs single-package** (workspace configs, multiple build targets)
2. **Frontend/backend split** (separate entry points, directory structure)
3. **Database and ORM** (from dependencies and infrastructure config)

### Step 4: Capture Conventions

Map conventions from `CLAUDE.md` (if present) into the `config.yaml` `conventions` section.

### Step 5: Create Directory Structure

Create the following directories and files:

```
openspec/
  config.yaml           # Project configuration and SDD rules
  specs/                # Source of truth for current system specifications
    .gitkeep
  changes/              # Active change proposals and artifacts
    .gitkeep
    archive/            # Completed and archived changes
      .gitkeep
```


### Step 6: Generate config.yaml

Generate `openspec/config.yaml` with the following sections:

```yaml
schema: spec-driven
version: "1.0"
generated_at: <ISO 8601 timestamp>

project:
  name: <detected from manifest or directory name>
  path: <absolute project path>
  type: <monorepo | single-package>

stack:
  language: <detected language>
  runtime: <detected runtime>
  frameworks: <detected frameworks list>

commands:
  typecheck: <detected build/type check command>
  lint: <detected lint command>
  lint_fix: <detected lint fix command>
  test: <detected test command>
  format_check: <detected format check command>
  format_fix: <detected format fix command>

contracts:
  # Auto-assembled from PARCER Contract blocks in each phase's SKILL.md.
  # Scan ~/.claude/skills/sdd/sdd-*/SKILL.md for ## PARCER Contract sections
  # and merge here. Re-run /sdd:init to regenerate.
```

The `commands` block is the most critical output — all downstream phases read it. Detect commands from `CLAUDE.md`, manifest scripts, Makefile targets, or ecosystem conventions.

### Step 6b: Assemble PARCER Contracts

Populate the `contracts` section by scanning `~/.claude/skills/sdd/sdd-*/SKILL.md` files. For each file with a `## PARCER Contract` section, extract the YAML block and merge it into `contracts:` keyed by phase name. Skip phases without contracts.

### Step 7: Present Summary

Present a markdown summary to the user, then STOP. Do not proceed automatically.

**On success, output:**

```markdown
## SDD Init Complete

**Project**: {project_name}
**Stack**: {stack_summary}
**Architecture**: {monorepo | single-package}

### Files Created
- `openspec/config.yaml` — full project configuration
- `openspec/specs/` — baseline spec directory
- `openspec/changes/` — change tracking directory

### Conventions Captured
- Source: {CLAUDE.md | inferred | none}
- {N} coding rules and {N} verification commands registered

{If warnings: ### ⚠ Warnings\n- {warning}\n}

**Next step**: Run `/sdd:explore <topic>` to investigate an area, or `/sdd:new <change-name> "<intent>"` to start a change.
```

If already initialized and `--force` was not set: output a short note that `openspec/` already exists and suggest `/sdd:new` to start a change.
If `--dry-run` was set: output what would be created without having written any files.

## Rules and Constraints

1. **Read-only analysis of the project** -- only create files inside `openspec/`.
2. **Never modify existing project files** -- no changes to `package.json`, `tsconfig.json`, etc.
3. **If `openspec/` already exists and `force` is false**, do not overwrite. Read and report state.
4. **Support monorepo detection** -- Bun workspaces, npm workspaces, pnpm workspaces, Turborepo, Nx.
5. **The `config.yaml` must capture ALL conventions from `CLAUDE.md`** if it exists. Do not skip any rules.
6. **Use absolute paths** in all output references.
7. **If `dry_run` is true**, present a summary describing what would be created without actually writing files.
8. **Timestamp all generated files** with ISO 8601 format.
9. **Never include secrets or environment variable values** in config.yaml -- only reference variable names.
10. **If detection is uncertain**, include a `warnings` list in the summary output explaining what could not be auto-detected.

## Error Handling

- If the project root does not exist: return `status: error` with message.
- If no `package.json`, `go.mod`, `pyproject.toml`, `Cargo.toml`, or equivalent is found: return `status: error` with message suggesting manual configuration.
- If file read fails: log the file path and continue with partial detection.
- All errors must include the phase name (`init`) and a human-readable message.

