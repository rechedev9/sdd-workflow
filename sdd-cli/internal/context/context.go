// Package context implements per-phase context assemblers.
// Each assembler loads the relevant SKILL.md + prior artifacts + source context,
// then writes assembled context to an io.Writer (stdout).
//
// Features:
// - Content-hash cache: skip re-assembly if input artifacts unchanged
// - Inline metrics: bytes, estimated tokens, duration on stderr
// - Size guard: reject if assembled context exceeds maxContextBytes
package context

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// Assembler is a function that writes assembled context to w.
type Assembler func(w io.Writer, p *Params) error

// Params holds everything an assembler needs.
type Params struct {
	ChangeDir   string
	ChangeName  string
	Description string
	ProjectDir  string
	Config      *config.Config
	SkillsPath  string
	Stderr    io.Writer      // for metrics output; nil = discard
	Verbosity int           // -1=quiet, 0=default, 1=verbose, 2=debug
	Broker    *events.Broker // event broker; nil = no events
}

// dispatchers maps phases to their assembler functions.
var dispatchers = map[state.Phase]Assembler{
	state.PhaseExplore: AssembleExplore,
	state.PhasePropose: AssemblePropose,
	state.PhaseSpec:    AssembleSpec,
	state.PhaseDesign:  AssembleDesign,
	state.PhaseTasks:   AssembleTasks,
	state.PhaseApply:   AssembleApply,
	state.PhaseReview:  AssembleReview,
	state.PhaseClean:   AssembleClean,
}

// Assemble resolves the phase and runs the appropriate assembler.
// Uses content-hash caching to skip assembly if inputs haven't changed.
// Emits events via p.Broker for metrics, caching, and stderr output.
// Enforces a size guard on assembled context.
func Assemble(w io.Writer, phase state.Phase, p *Params) error {
	fn, ok := dispatchers[phase]
	if !ok {
		return fmt.Errorf("no assembler for phase: %s", phase)
	}

	phaseStr := string(phase)
	start := time.Now()

	// Try cache first.
	if cached, ok := tryCachedContext(p.ChangeDir, phaseStr, p.SkillsPath); ok {
		size := len(cached)
		w.Write(cached)

		p.Broker.Emit(events.Event{
			Type: events.CacheHit,
			Payload: events.CacheHitPayload{
				Phase: phaseStr,
				Bytes: size,
			},
		})

		p.Broker.Emit(events.Event{
			Type: events.PhaseAssembled,
			Payload: events.PhaseAssembledPayload{
				Phase:      phaseStr,
				Bytes:      size,
				Tokens:     estimateTokens(size),
				Cached:     true,
				DurationMs: time.Since(start).Milliseconds(),
				ChangeDir:  p.ChangeDir,
				SkillsPath: p.SkillsPath,
			},
		})

		return nil
	}

	p.Broker.Emit(events.Event{
		Type:    events.CacheMiss,
		Payload: events.CacheMissPayload{Phase: phaseStr},
	})

	// Assemble into buffer for caching + size check.
	var buf bytes.Buffer
	if err := fn(&buf, p); err != nil {
		return err
	}

	size := buf.Len()

	// Size guard.
	if size > maxContextBytes {
		return fmt.Errorf("context too large: %s (%d bytes, ~%dK tokens) exceeds limit of %s (~%dK tokens)",
			formatBytes(size), size, estimateTokens(size)/1000,
			formatBytes(maxContextBytes), estimateTokens(maxContextBytes)/1000)
	}

	// Write to output.
	content := buf.Bytes()
	w.Write(content)

	p.Broker.Emit(events.Event{
		Type: events.PhaseAssembled,
		Payload: events.PhaseAssembledPayload{
			Phase:      phaseStr,
			Bytes:      size,
			Tokens:     estimateTokens(size),
			Cached:     false,
			DurationMs: time.Since(start).Milliseconds(),
			ChangeDir:  p.ChangeDir,
			SkillsPath: p.SkillsPath,
			Content:    content,
		},
	})

	return nil
}

// AssembleConcurrent assembles multiple phases in parallel and writes
// results to w in the order of the input slice (deterministic output).
// Used for spec+design which can run concurrently after propose.
// Inspired by sag's bounded semaphore concurrency pattern.
func AssembleConcurrent(w io.Writer, phases []state.Phase, p *Params) error {
	if len(phases) == 0 {
		return nil
	}
	if len(phases) == 1 {
		return Assemble(w, phases[0], p)
	}

	type result struct {
		data []byte
		err  error
	}

	results := make([]result, len(phases))
	var wg sync.WaitGroup

	for i, phase := range phases {
		wg.Add(1)
		go func(idx int, ph state.Phase) {
			defer wg.Done()
			var buf bytes.Buffer
			err := Assemble(&buf, ph, p)
			results[idx] = result{data: buf.Bytes(), err: err}
		}(i, phase)
	}

	wg.Wait()

	// Write successes in order, collect errors.
	// Partial output is intentional — better than nothing for the sub-agent.
	var errs []string
	for i, r := range results {
		if r.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", phases[i], r.err))
			continue
		}
		w.Write(r.data)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d/%d phases failed: %s", len(errs), len(phases), strings.Join(errs, "; "))
	}
	return nil
}

// loadSkill reads a SKILL.md file from the skills directory.
func loadSkill(skillsPath, phaseName string) ([]byte, error) {
	path := filepath.Join(skillsPath, phaseName, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load skill %s: %w", phaseName, err)
	}
	return data, nil
}

// loadArtifact reads an artifact file from the change directory.
func loadArtifact(changeDir, filename string) ([]byte, error) {
	path := filepath.Join(changeDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load artifact %s: %w", filename, err)
	}
	return data, nil
}

// writeSection writes a labeled section to the output.
func writeSection(w io.Writer, label string, content []byte) {
	fmt.Fprintf(w, "\n--- %s ---\n\n", label)
	w.Write(content)
	fmt.Fprintln(w)
}

// writeSectionStr writes a labeled section with string content.
func writeSectionStr(w io.Writer, label, content string) {
	writeSection(w, label, []byte(content))
}
