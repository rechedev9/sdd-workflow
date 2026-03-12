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


### Step 5c: Detect Memory Capabilities

Check if the persistent memory RAG server is available in the current environment:

1. **Probe** — Attempt to call `mem_stats` (a lightweight read-only memory tool). If it succeeds and reports healthy backends, memory is available.
2. **Record result** — Set `capabilities.memory_enabled` in config.yaml:
   - `true` if `mem_stats` responded successfully with healthy Qdrant and Ollama
   - `false` if the tool call failed, timed out, or the tool doesn't exist
3. **Log** — Note in the summary warnings if memory is unavailable: `"Memory RAG server not detected. SDD will run in Ephemeral Mode (no cross-session memory)."`

This flag drives conditional behavior in all downstream phases: when `true`, phases use full memory integration (EET, learning saves, context recovery). When `false`, phases skip all `mem_*` calls and use more aggressive local fallbacks.

### Step 6: Generate config.yaml

The `config.yaml` must follow this structure:

```yaml
schema: spec-driven
version: "1.0"
generated_at: <ISO 8601 timestamp>

project:
  name: <from package.json name or directory name>
  path: <absolute project path>
  type: <monorepo | single-package>

stack:
  runtime: <bun | node | deno | go | python | rust>
  language: <typescript | javascript | go | python | rust>
  language_version: <from tsconfig target or runtime version>
  frameworks:
    frontend: <react | vue | svelte | none>
    backend: <elysia | express | fastify | hono | none>
    testing: <bun:test | vitest | jest | none>
  database: <postgresql | mysql | sqlite | none>
  orm: <drizzle | prisma | typeorm | none>

architecture:
  pattern: <monorepo | single-package>
  workspaces: <list of workspace paths if monorepo>
  entry_points:
    frontend: <path to frontend entry>
    backend: <path to backend entry>

capabilities:
  memory_enabled: <true | false — detected in Step 5c>

commands:
  package_manager: <bun | npm | pnpm | yarn — detected from lockfile>
  typecheck: <e.g., "bun run typecheck" | "pnpm run typecheck:all">
  lint: <e.g., "bun run lint" | "pnpm run check:all">
  lint_fix: <e.g., "bun run lint:fix" | "pnpm --filter <pkg> lint:fix">
  test: <e.g., "bun test" | "pnpm test:all">
  format_check: <e.g., "bun run format:check" | "pnpm prettier --check">
  format_fix: <e.g., "bun run prettier --write" | "pnpm prettier --write">

conventions:
  type_strictness:
    banned: <list of banned patterns from CLAUDE.md>
    allowed: <list of allowed patterns>
    test_only: <list of test-only patterns>
  error_handling:
    pattern: <result | throw | either>
    result_type_path: <path to Result type if applicable>
  testing:
    runner: <bun:test | vitest | jest>
    pattern: <describe-it | describe-test>
    file_suffix: <.test.ts | .spec.ts>
    location: <colocated | __tests__>
  file_organization:
    max_lines_warning: <number>
    max_lines_error: <number>
    max_nesting_depth: <number>
  code_style:
    async_preference: <async-await | then-chains>
    immutability: <prefer-immutable | mutable>
    no_console_log: <boolean>

phases:
  proposal:
    required_sections:
      - intent
      - scope
      - approach
      - affected_areas
      - risks
      - rollback_plan
      - dependencies
      - success_criteria
  specs:
    keywords: RFC2119
    scenario_format: given-when-then
    min_scenarios_per_requirement: 1
  design:
    required_sections:
      - technical_approach
      - architecture_decisions
      - data_flow
      - file_changes
      - interfaces
      - testing_strategy
  tasks:
    ordering: bottom-up
    ordering_hint: ""
    task_format: "N.M Action - file, change"
  apply:
    batch_size: 1
    verify_after_each: true
  review:
    checklist:
      - type_safety
      - error_handling
      - test_coverage
      - naming_conventions
  verify:
    # Note: build commands are now in top-level `commands:` block.
    # All phases (apply, verify, clean) read from there.
  clean:
    merge_specs: true
    archive_changes: true

contracts:
  # AUTO-ASSEMBLED from PARCER Contract blocks in each phase's SKILL.md
  # sdd-init scans ~/.claude/skills/sdd/sdd-*/SKILL.md for ## PARCER Contract sections
  # and merges them here. Do not edit manually — re-run /sdd:init to regenerate.
```

### Step 6b: Assemble PARCER Contracts

The `contracts` section in `config.yaml` is NOT hardcoded. It is dynamically assembled from each phase's self-declared contract:

1. **Scan** — List all directories matching `~/.claude/skills/sdd/sdd-*/`
2. **Extract** — For each `SKILL.md` found, search for a `## PARCER Contract` section containing a YAML code block
3. **Parse** — Extract the `phase`, `preconditions`, and `postconditions` fields from each YAML block
4. **Merge** — Assemble all contracts into the `contracts:` section of `openspec/config.yaml`:
   ```yaml
   contracts:
     explore:
       preconditions: [...]
       postconditions: [...]
     propose:
       preconditions: [...]
       postconditions: [...]
     # ... one entry per phase with a PARCER Contract
   ```
5. **Skip phases without contracts** — If a SKILL.md has no `## PARCER Contract` section, do not add an entry (the phase has no validation requirements)
6. **Log** — Include in `phaseSpecificData.warnings` any phases found without PARCER contracts

This makes the SDD ecosystem **plug-and-play**: adding a new phase (e.g., `sdd-security-audit`) only requires creating its SKILL.md with a `## PARCER Contract` block. The next `/sdd:init` run will auto-detect and register it.

### Step 7: Present Summary

Present a markdown summary to the user, then STOP. Do not proceed automatically.

If `capabilities.quality_tracking` is enabled in `openspec/config.yaml`, append one line to `openspec/changes/.quality-init.jsonl` (or skip — init has no changeName):
```json
{ "phase": "init", "timestamp": "<ISO 8601>", "agentStatus": "SUCCESS", "stack": "<stack_summary>", "warnings": [] }
```

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

