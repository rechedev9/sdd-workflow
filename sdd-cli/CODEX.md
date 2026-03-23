# CODEX.md — SDD CLI Integration Guide for Codex

## Token Budget Rules

The SDD pipeline produces ~100K tokens of context across 8 phases. Without discipline, a single-session run balloons to millions of tokens due to conversational context accumulation.

### Mandatory

1. **Use `sdd context --compact` for phases after tasks.** The `--compact` flag reduces specs to headings + MUST/SHOULD lines and design to decisions only. Full artifacts are only needed for the phase that produces them.

2. **Never re-read artifact files that `sdd context` already includes.** The context output contains specs/, design.md, tasks.md etc. Reading them separately doubles token consumption.

3. **One session per phase.** After promoting with `sdd write <name> <phase>`, end the session. Start a new one for the next phase. This prevents O(n^2) context growth from conversation history accumulation.

4. **Use `/sdd-continue` to auto-route phases.** Do not manually decide which phase to run — the CLI knows the state machine.

### Phase-specific

| Phase | Context flag | Notes |
|-------|-------------|-------|
| explore | (default) | Full context needed for codebase investigation |
| propose | (default) | Needs full exploration.md |
| spec | (default) | Needs full proposal.md |
| design | (default) | Needs full proposal.md + specs/ |
| tasks | `--compact` | Specs/design are reference only |
| apply | `--compact` | Only current task matters; design/specs for reference |
| review | `--compact` | Compact specs/design; diff is the primary input |
| clean | `--compact` | Verify report is primary; design/specs for reference |

### What NOT to do

- Do NOT run the entire pipeline in a single session
- Do NOT read `openspec/changes/*/` files directly — use `sdd context`
- Do NOT accumulate build/test output in conversation — check results, then discard
- Do NOT re-run `sdd context` multiple times for the same phase in one session
