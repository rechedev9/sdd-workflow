# SDD Ship — Create PR for Trunk-Based Merge

## Role

You are shipping a completed SDD change as a pull request. The Go CLI does the heavy lifting — your job is to validate preconditions, invoke the command, and present results.

## Preconditions

- Change must have `clean` phase completed
- `state.json` must have a `base_ref` (set by `sdd new`)
- `gh` CLI must be installed and authenticated
- No remote branch `sdd/<name>` may already exist

## Execution

### Step 1: Validate

```bash
sdd status <name>
```

Confirm `current_phase` is `ship`. If not, tell the user which phases remain.

### Step 2: Ship

```bash
sdd ship <name>
```

The Go command will:
1. Create branch `sdd/<name>` from current HEAD
2. Push to origin
3. Open a PR via `gh pr create` with summary from proposal.md and design.md
4. Write `ship-report.md`
5. Advance state to `archive`
6. Return to master

### Step 3: Present

Parse the JSON output. Show:
- PR URL as a clickable link
- Branch name
- Number of files in the PR
- Next steps: review on GitHub, merge, then `/sdd-archive <name>`

### Step 4: Suggest

After merge:
- `/sdd-archive <name>` — merges delta specs into main specs, extracts ADRs, captures learnings, moves folder to archive

## Flags

| Flag | Effect |
|------|--------|
| `--dry-run` | Show what would happen without creating branch/PR |
| `--title <t>` | Override PR title (default: `feat(<name>): <description>`) |

## Error Recovery

| Error | Action |
|-------|--------|
| `gh not found` | Tell user: `install from https://cli.github.com` |
| `gh not authenticated` | Tell user: `gh auth login` |
| Branch exists | Tell user: `git push origin --delete sdd/<name>` |
| Push failed | Check git remote config, network |
| PR creation failed | Branch is auto-cleaned up; retry after fixing |
