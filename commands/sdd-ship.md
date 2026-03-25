# /sdd-ship — Ship Change as PR

## Arguments
$ARGUMENTS — Change name (required). Optional flags: `--dry-run`, `--title <title>`.

## Execution

### Step 1: Run ship

```bash
sdd ship <name> [--dry-run] [--title <title>]
```

Parse JSON output.

### Step 2: Present results

Show:
1. Branch name: `sdd/<name>`
2. PR URL (clickable link)
3. Files included
4. Next steps

### Step 3: Suggest next actions

After the PR is merged on GitHub:
- `/sdd-archive <name>` to merge delta specs and clean up
- The squash merge commit on master is the permanent record

### Error handling

- **gh not installed:** tell user to install from https://cli.github.com
- **gh not authenticated:** tell user to run `gh auth login`
- **Branch already exists:** tell user to delete it first: `git push origin --delete sdd/<name>`
- **Prerequisites not met:** tell user which phases are still pending
