// Package phase defines the canonical Phase descriptor and a Registry
// that replaces scattered package-level maps across state, artifacts,
// and context. Each phase is self-describing: prerequisites, artifact
// filename, cache inputs, cache TTL, and assembler function.
//
// Import constraint: this package imports only config and events (both
// leaves). It must NOT import state, artifacts, or context to avoid
// import cycles.
package phase

import (
	"io"
	"slices"
	"sync/atomic"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
)

// AssemblerParams holds everything an assembler function needs.
// Equivalent to the former context.Params; context.Params becomes
// a type alias for this struct.
type AssemblerParams struct {
	ChangeDir   string
	ChangeName  string
	Description string
	ProjectDir  string
	Config      *config.Config
	SkillsPath  string
	Broker      *events.Broker // event broker; nil = no events
}

// Assembler is the function type for per-phase context assembly.
type Assembler func(w io.Writer, p *AssemblerParams) error

// Phase is the canonical descriptor for one SDD pipeline phase.
// Name uses plain string (not state.Phase) to avoid import cycles.
type Phase struct {
	Name          string        // matches state.Phase constant values
	Prerequisites []string      // names of phases that must be completed first
	NextPhases    []string      // names of phases this phase can transition to
	ArtifactFile  string        // final artifact filename or dir
	RecoverSkip   bool          // true = Recover() skips this phase
	CacheInputs   []string      // artifact paths that invalidate the cache
	CacheTTL      time.Duration // 0 = no TTL
	Assemble      Assembler     // nil for verify, archive
}

// Registry holds the ordered slice of Phase descriptors.
// Order = pipeline position; used by AllPhases() and nextReady().
// Read-only after first Get()/All()/AllNames() call.
type Registry struct {
	phases []Phase
	byName map[string]int // name → index in phases; populated by Register
	sealed atomic.Bool
}

// Register appends a Phase to the registry.
// Panics if called after the registry is sealed.
// Panics if p.Name is empty or already registered.
func (r *Registry) Register(p Phase) {
	if r.sealed.Load() {
		panic("phase: Register called on sealed registry")
	}
	if p.Name == "" {
		panic("phase: Register called with empty Name")
	}
	if r.byName == nil {
		r.byName = make(map[string]int, 16)
	}
	if _, dup := r.byName[p.Name]; dup {
		panic("phase: duplicate registration: " + p.Name)
	}
	// Sort CacheInputs once at registration so inputHash can skip the alloc+sort.
	if len(p.CacheInputs) > 1 {
		slices.Sort(p.CacheInputs)
	}
	r.byName[p.Name] = len(r.phases)
	r.phases = append(r.phases, p)
}

// SetAssembler sets the Assemble function for a named phase.
// Used by context package init() to wire assemblers without import cycles.
// Panics if sealed or if the phase name is not found.
func (r *Registry) SetAssembler(name string, fn Assembler) {
	if r.sealed.Load() {
		panic("phase: SetAssembler called on sealed registry")
	}
	if i, ok := r.byName[name]; ok {
		r.phases[i].Assemble = fn
		return
	}
	panic("phase: SetAssembler called for unknown phase: " + name)
}

// Get returns the Phase descriptor for the given name.
// Seals the registry on first call.
func (r *Registry) Get(name string) (Phase, bool) {
	r.sealed.Store(true)
	if i, ok := r.byName[name]; ok {
		return r.phases[i], true
	}
	return Phase{}, false
}

// All returns a copy of the ordered phase slice.
// Seals the registry.
func (r *Registry) All() []Phase {
	r.sealed.Store(true)
	out := make([]Phase, len(r.phases))
	copy(out, r.phases)
	return out
}

// AllNames returns phase names in pipeline order.
// Seals the registry.
func (r *Registry) AllNames() []string {
	r.sealed.Store(true)
	names := make([]string, len(r.phases))
	for i, p := range r.phases {
		names[i] = p.Name
	}
	return names
}

// DefaultRegistry is the package-level singleton used by all internal packages.
var DefaultRegistry = &Registry{}
