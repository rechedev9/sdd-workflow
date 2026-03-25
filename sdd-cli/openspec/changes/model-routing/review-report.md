# Review: Model Routing por Fase

**Change**: model-routing
**Phase**: review
**Verdict**: PASS

## Diff Summary

8 files changed, 273 insertions(+), 16 deletions(-)

| File | Change | Lines |
|------|--------|-------|
| `internal/config/types.go` | Added `Models` struct + field | +8 |
| `internal/config/config.go` | Added `ModelFor`, validation, wiring | +39 |
| `internal/config/config_test.go` | 8 new tests for model routing config | +118 |
| `internal/context/context.go` | Model directive injection in `Assemble()` | +9 |
| `internal/context/context_test.go` | 4 new tests for directive + events | +75 |
| `internal/events/broker.go` | Added `Model` field to payload | +1 |
| `openspec/config.yaml` | Added `models` section | +7 |
| `internal/cli/cmd_ship_test.go` | Unrelated reformatting (gofumpt) | +16/-16 |

## Checklist

- [x] **Backward compatible**: Empty `Models{}` → no directive, no validation error
- [x] **Validation**: Invalid model names and unknown phases rejected at `Load()` time
- [x] **Injection point**: Directive prepended outside cache layer — cache stays valid
- [x] **Telemetry**: Model included in all 3 `PhaseAssembled` event emit paths
- [x] **Test coverage**: 12 new tests covering fallback chain, validation, directive presence/absence, event payload
- [x] **No import cycles**: `config` package uses local `phaseNames` set, no `phase` import
- [x] **`make check` passes**: 0 lint issues, all tests green

## Issues Found

None. Implementation matches spec and design.

## Key Code References

- Model struct definition: `internal/config/types.go:36`
- ModelFor method: `internal/config/config.go:30`
- Validation: `internal/config/config.go:40`
- Directive injection: `internal/context/context.go:74`
- Event payload field: `internal/events/broker.go:39`

## Risks Reviewed

- Map iteration in `validateModels` (`internal/config/config.go:45`) is non-deterministic — error messages for multiple invalid entries may vary in order. Acceptable for validation errors.
- `phaseNames` in `internal/config/config.go:24` is a static duplicate of phase registry names. If a new phase is added to the registry but not to `phaseNames`, validation will reject it. Mitigated by: phases change very rarely, and the error message is clear.
