# Design: Model Routing por Fase

**Change**: model-routing
**Phase**: design

## Architecture

```
config.yaml                    context.Assemble()
┌──────────┐                  ┌────────────────────────┐
│ models:  │   config.Load()  │ 1. model = ModelFor(ph)│
│  default │ ──────────────►  │ 2. write directive     │
│  phases: │                  │ 3. write content       │
│    ...   │                  │ 4. emit event w/model  │
└──────────┘                  └────────────────────────┘
                                        │
                                        ▼ stdout
                              <!-- sdd:model=opus -->
                              --- SKILL ---
                              ...
```

## File-Level Changes

### 1. `internal/config/types.go`

Add after `Capabilities`:

```go
// Models configures per-phase LLM model routing.
// Zero value = no model directives emitted (backward compatible).
type Models struct {
	Default string            `yaml:"default" json:"default"`
	Phases  map[string]string `yaml:"phases"  json:"phases,omitempty"`
}
```

Add `Models` field to `Config` struct:

```go
Models       Models       `yaml:"models"         json:"models"`
```

### 2. `internal/config/config.go`

Add valid models set and `ModelFor` method:

```go
var validModels = map[string]bool{
	"opus":   true,
	"sonnet": true,
	"haiku":  true,
}

// ModelFor returns the model configured for the given phase.
// Returns phase-specific override if set, else default, else "".
func (c *Config) ModelFor(phase string) string {
	if m, ok := c.Models.Phases[phase]; ok {
		return m
	}
	return c.Models.Default
}
```

Add validation in `Load()` after YAML unmarshal:

```go
if err := validateModels(cfg.Models); err != nil {
	return nil, fmt.Errorf("invalid models config: %w", err)
}
```

Validation function:

```go
func validateModels(m Models) error {
	if m.Default != "" && !validModels[m.Default] {
		return fmt.Errorf("unknown default model: %q (valid: opus, sonnet, haiku)", m.Default)
	}
	for phase, model := range m.Phases {
		if !validModels[model] {
			return fmt.Errorf("unknown model %q for phase %q (valid: opus, sonnet, haiku)", model, phase)
		}
		if _, ok := phaseNames[phase]; !ok {
			return fmt.Errorf("unknown phase %q in models.phases", phase)
		}
	}
	return nil
}
```

Note: `phaseNames` needs to be a set of valid phase name strings. Since `config` cannot import `phase` (import cycle), we define the set directly in config as a package-level var:

```go
var phaseNames = map[string]bool{
	"explore": true, "propose": true, "spec": true, "design": true,
	"tasks": true, "apply": true, "review": true, "verify": true,
	"clean": true, "ship": true, "archive": true,
}
```

### 3. `internal/context/context.go`

In `Assemble()`, add model directive injection. The directive must be written **outside** the cache — prepended to output regardless of cache hit/miss:

```go
func Assemble(w io.Writer, ph state.Phase, p *Params) error {
	desc, ok := phase.DefaultRegistry.Get(string(ph))
	if !ok || desc.Assemble == nil {
		return fmt.Errorf("no assembler for phase: %s", ph)
	}

	phaseStr := string(ph)
	model := p.Config.ModelFor(phaseStr)  // NEW
	start := time.Now()

	// Write model directive before any content.
	if model != "" {                                           // NEW
		io.WriteString(w, "<!-- sdd:model="+model+" -->\n\n")  // NEW
	}                                                          // NEW

	// ... rest of function unchanged ...
```

The same injection applies to both compact and cached paths — the directive is written first, then content follows.

For `PhaseAssembled` events, add `Model: model` to all three emit sites (compact, cached, fresh).

### 4. `internal/events/broker.go`

Add `Model` field to `PhaseAssembledPayload`:

```go
type PhaseAssembledPayload struct {
	Phase      string `json:"phase"`
	Bytes      int    `json:"bytes"`
	Tokens     int    `json:"tokens"`
	Cached     bool   `json:"cached"`
	DurationMs int64  `json:"duration_ms"`
	ChangeDir  string `json:"change_dir,omitempty"`
	SkillsPath string `json:"skills_path,omitempty"`
	Content    []byte `json:"-"`
	InputHash  string `json:"input_hash,omitempty"`
	Model      string `json:"model,omitempty"`   // NEW
}
```

### 5. `openspec/config.yaml`

Add models section to project config:

```yaml
models:
  default: sonnet
  phases:
    propose: opus
    spec: opus
    design: opus
    review: opus
```

## Cache Interaction

Model directive is **not cached** — it's prepended to output at `Assemble()` level, outside the cache layer. This means:

- Changing `models` config takes effect immediately (no cache bust needed)
- Cache hash remains stable (model config not part of input hash)
- Directive bytes are not counted in the cached `Bytes` metric (acceptable; directive is ~30 bytes)

## Test Plan

1. **Config round-trip**: Load config with `models` section, verify `ModelFor()` returns correct values
2. **Validation**: Invalid model name → error. Unknown phase → error. Empty models → no error.
3. **Directive injection**: Assemble with model config → output starts with `<!-- sdd:model=X -->`
4. **No directive**: Assemble without model config → output starts with `--- SKILL ---`
5. **Fallback chain**: Phase override > default > empty
6. **Event payload**: PhaseAssembled event includes model field
