# Spec: Model Routing por Fase

**Change**: model-routing
**Phase**: spec

## Data Structures

### Models (new struct in `config/types.go`)

```go
// Models configures per-phase LLM model routing.
type Models struct {
    Default string            `yaml:"default" json:"default"`
    Phases  map[string]string `yaml:"phases"  json:"phases,omitempty"`
}
```

### Config (modified — add Models field)

```go
type Config struct {
    // ... existing fields ...
    Models Models `yaml:"models" json:"models"`
}
```

### PhaseAssembledPayload (modified — add Model field)

```go
type PhaseAssembledPayload struct {
    // ... existing fields ...
    Model string `json:"model,omitempty"`
}
```

## Functions

### `Config.ModelFor(phase string) string`

```
Input:  phase name (e.g. "propose", "apply")
Output: model name or empty string

Logic:
  1. If c.Models.Phases[phase] exists → return it
  2. If c.Models.Default != "" → return it
  3. Return ""
```

### `validateModels(m Models) error`

```
Input:  Models struct
Output: error if any model name is invalid

Valid models: {"opus", "sonnet", "haiku"}

Logic:
  1. If m.Default != "" and m.Default not in validModels → error
  2. For each (phase, model) in m.Phases:
     a. If model not in validModels → error
     b. If phase not in phase.DefaultRegistry.AllNames() → error
  3. Return nil
```

### `Assemble()` modification (context.go)

```
Before writing assembled content (both cached and fresh paths):
  1. model := p.Config.ModelFor(phaseStr)
  2. If model != "":
     a. Write "<!-- sdd:model={model} -->\n\n" to w
  3. Include model in PhaseAssembled event payload
```

## YAML Config Schema

```yaml
# openspec/config.yaml — models section (optional)
models:
  default: sonnet          # fallback for phases not listed below
  phases:                  # per-phase overrides (all optional)
    explore: sonnet
    propose: opus
    spec: opus
    design: opus
    tasks: sonnet
    apply: sonnet
    review: opus
    clean: sonnet
```

## Validation Rules

1. Model names must be one of: `opus`, `sonnet`, `haiku`
2. Phase names in `models.phases` must match registered phase names
3. Empty `models` section = no directives emitted (backward compatible)
4. Empty `default` with non-empty `phases` = directive only for listed phases

## Edge Cases

| Case | Behavior |
|------|----------|
| No `models` section in config | Zero-value Models struct; ModelFor returns "" for all phases; no directive |
| `default` set, no `phases` | All phases get default model |
| Phase in `phases` overrides `default` | Phase-specific wins |
| Unknown model name | `Load()` returns validation error |
| Unknown phase name in `phases` map | `Load()` returns validation error |
| `models.phases` is nil map | ModelFor safely returns "" (map lookup on nil returns zero value) |
