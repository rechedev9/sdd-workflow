package artifacts

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// ErrValidation indicates that an artifact failed content validation.
var ErrValidation = errors.New("artifact validation failed")

// rule checks one requirement against artifact content.
type rule struct {
	name  string
	check func([]byte) bool
}

var fileLineRef = regexp.MustCompile(`\w+\.\w+:\d+`)

var phaseRules = map[state.Phase][]rule{
	state.PhaseExplore: {
		{name: "## Current State heading", check: containsStr("## Current State")},
		{name: "## Relevant Files heading", check: containsStr("## Relevant Files")},
	},
	state.PhasePropose: {
		{name: "## Intent heading", check: containsStr("## Intent")},
		{name: "## Scope heading", check: containsStr("## Scope")},
	},
	state.PhaseSpec: {
		{name: "at least one ## heading", check: containsStr("## ")},
	},
	state.PhaseDesign: {
		{name: "at least one ## heading", check: containsStr("## ")},
	},
	state.PhaseTasks: {
		{name: "at least one task checkbox", check: containsStr("- [")},
	},
	state.PhaseApply: {
		{name: "at least one task checkbox", check: containsStr("- [")},
	},
	state.PhaseReview: {
		{name: "at least one ## heading", check: containsStr("## ")},
		{name: "file:line reference (e.g. main.go:42)", check: matchesRegex(fileLineRef)},
		{name: "verdict (PASS, FAIL, APPROVED, or REJECTED)", check: containsAny("PASS", "FAIL", "APPROVED", "REJECTED")},
	},
	// verify, clean, archive — no content rules (or minimal)
	state.PhaseClean: {
		{name: "at least one ## heading", check: containsStr("## ")},
	},
}

// Validate checks that content satisfies all rules for the given phase.
// Returns nil if valid or if the phase has no rules.
func Validate(phase state.Phase, content []byte) error {
	rules, ok := phaseRules[phase]
	if !ok || len(rules) == 0 {
		return nil
	}

	missing := make([]string, 0, len(rules))
	for _, r := range rules {
		if !r.check(content) {
			missing = append(missing, r.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s: missing %s", ErrValidation, phase, strings.Join(missing, ", "))
}

// ValidatePending validates the pending artifact at path for the given phase.
// Directory-backed phases validate every markdown file under the directory.
func ValidatePending(phase state.Phase, path string) error {
	if !isDirectoryArtifact(phase) {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read pending: %w", err)
		}
		return Validate(phase, data)
	}

	files, err := collectRegularFiles(path)
	if err != nil {
		return fmt.Errorf("walk pending: %w", err)
	}

	mdCount := 0
	failures := make([]string, 0)
	for _, file := range files {
		if filepath.Ext(file) != ".md" {
			continue
		}
		mdCount++
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read pending: %w", err)
		}
		if err := Validate(phase, data); err != nil {
			rel, relErr := filepath.Rel(path, file)
			if relErr != nil {
				rel = file
			}
			failures = append(failures, fmt.Sprintf("%s (%v)", rel, err))
		}
	}
	if mdCount == 0 {
		return fmt.Errorf("%w: %s: missing markdown files", ErrValidation, phase)
	}
	if len(failures) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(failures, "; "))
	}
	return nil
}

func containsStr(s string) func([]byte) bool {
	b := []byte(s)
	return func(content []byte) bool {
		return bytes.Contains(content, b)
	}
}

func containsAny(options ...string) func([]byte) bool {
	bs := make([][]byte, len(options))
	for i, opt := range options {
		bs[i] = []byte(opt)
	}
	return func(content []byte) bool {
		for _, b := range bs {
			if bytes.Contains(content, b) {
				return true
			}
		}
		return false
	}
}

func matchesRegex(re *regexp.Regexp) func([]byte) bool {
	return func(content []byte) bool {
		return re.Match(content)
	}
}
