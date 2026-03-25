# Apply: Model Routing por Fase

**Change**: model-routing
**Phase**: apply

## Summary of Changes

### Files Modified

1. **`internal/config/types.go`** — Added `Models` struct with `Default` + `Phases` map. Added `Models` field to `Config`.

2. **`internal/config/config.go`** — Added `validModels` set, `phaseNames` set, `ModelFor()` method, `validateModels()` function. Wired validation into `Load()`.

3. **`internal/events/broker.go`** — Added `Model` field to `PhaseAssembledPayload`.

4. **`internal/context/context.go`** — Resolves model via `Config.ModelFor(phase)`, prepends `<!-- sdd:model=X -->` directive before all content (outside cache layer). Includes model in all 3 `PhaseAssembled` event emits.

5. **`openspec/config.yaml`** — Added `models` section with `default: sonnet` and phase overrides for propose, spec, design, review.

### Tests Added

6. **`internal/config/config_test.go`** — 8 new tests: ModelFor fallback chain (phase override, default only, empty), validateModels (valid, invalid default, invalid phase model, unknown phase, empty), Load with models, Load with invalid model.

7. **`internal/context/context_test.go`** — 4 new tests: directive present with override, directive with default, no directive when unconfigured, model field in PhaseAssembled event.

## Task Completion

- [x] T1: Add Models struct and Config field
- [x] T2: Add ModelFor method and validation
- [x] T3: Add Model field to PhaseAssembledPayload
- [x] T4: Inject model directive in Assemble()
- [x] T5: Update project config
- [x] T6: Tests

## Verification

- `make check` passes (0 lint issues, all tests green)
- Smoke test: `sdd context model-routing explore` → `<!-- sdd:model=sonnet -->`
- Smoke test: `sdd context model-routing propose` → `<!-- sdd:model=opus -->`
