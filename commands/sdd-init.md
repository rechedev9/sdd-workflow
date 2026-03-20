# /sdd-init — Bootstrap Spec-Driven Development

## Arguments
$ARGUMENTS — Optional: `--force` to reinitialize an existing project.

## Execution

### Step 1: Run sdd init

```bash
sdd init $ARGUMENTS
```

### Step 2: Present results

Parse JSON from stdout. Show:
1. Detected tech stack (language, build tool, manifests)
2. Created directory structure
3. Config.yaml location
4. Suggested next step: `/sdd-new <change-name> <description>`

If failed, show stderr JSON error and suggest fixes (e.g., "no manifest found — create a go.mod/package.json first").
