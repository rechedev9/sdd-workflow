# /sdd-archive — Close Completed Change

## Arguments
$ARGUMENTS — Change name (required).

## Execution

### Step 1: Run archive

```bash
sdd archive <name>
```

### Step 2: Present results

Parse JSON output. Show:
1. Archive location: full path
2. Manifest: artifact count, completed phases
3. Change summary: name, description, key artifacts preserved

### Step 3: Suggest next actions

- Commit and create PR for this change
- `/sdd-new <name>` — Start the next change
