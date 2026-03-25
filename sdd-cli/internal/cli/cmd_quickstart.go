package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runQuickstart(args []string, stdout io.Writer, stderr io.Writer) error {
	var specPath string
	var positional []string

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--spec":
			if i+1 >= len(args) {
				return errs.Usage("--spec requires a path argument")
			}
			specPath = args[i+1]
			i++ // consume next
		case !strings.HasPrefix(args[i], "-"):
			positional = append(positional, args[i])
		default:
			return errUnknownFlag(args[i])
		}
	}

	if len(positional) < 2 || specPath == "" {
		return errs.Usage("usage: sdd quickstart <name> \"<description>\" --spec <path>")
	}

	name := positional[0]
	description := positional[1]

	if err := validateChangeName(name); err != nil {
		return errs.Usage(err.Error())
	}

	// Validate spec file exists.
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return errs.WriteError(stderr, "quickstart", fmt.Errorf("read spec file: %w", err))
	}

	projectRoot, err := getProjectRoot(stderr, "quickstart")
	if err != nil {
		return err
	}

	// Ensure config exists.
	configPath := openspecConfig(projectRoot)
	if _, err := config.Load(configPath); err != nil {
		return errs.WriteError(stderr, "quickstart", fmt.Errorf("load config (run 'sdd init' first): %w", err))
	}

	// Create change directory structure.
	changeDir := filepath.Join(openspecChanges(projectRoot), name)
	specsDir := filepath.Join(changeDir, "specs")
	for _, d := range []string{changeDir, specsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return errs.WriteError(stderr, "quickstart", fmt.Errorf("create directory: %w", err))
		}
	}

	// Write the spec file as the design artifact (it's the source of truth).
	specBaseName := filepath.Base(specPath)
	type artifact struct {
		path    string
		content []byte
	}
	artifacts := []artifact{
		{filepath.Join(changeDir, "exploration.md"), []byte(fmt.Sprintf("# Exploration: %s\n\nFast-forwarded via `sdd quickstart`. See design spec for details.\n", name))},
		{filepath.Join(changeDir, "proposal.md"), []byte(fmt.Sprintf("# Proposal: %s\n\n%s\n\nFast-forwarded via `sdd quickstart`. See design spec for details.\n", name, description))},
		{filepath.Join(specsDir, specBaseName), specData},
		{filepath.Join(changeDir, "design.md"), specData},
		{filepath.Join(changeDir, "tasks.md"), []byte(fmt.Sprintf("# Tasks: %s\n\nDerived from spec. See design spec for task breakdown.\n\nFast-forwarded via `sdd quickstart`.\n", name))},
	}

	for _, a := range artifacts {
		if err := os.WriteFile(a.path, a.content, 0o644); err != nil {
			return errs.WriteError(stderr, "quickstart", fmt.Errorf("write artifact: %w", err))
		}
	}

	// Build state with explore→propose→spec→design→tasks completed, current=apply.
	now := time.Now().UTC()
	st := state.NewState(name, description)
	st.Phases[state.PhaseExplore] = state.StatusCompleted
	st.Phases[state.PhasePropose] = state.StatusCompleted
	st.Phases[state.PhaseSpec] = state.StatusCompleted
	st.Phases[state.PhaseDesign] = state.StatusCompleted
	st.Phases[state.PhaseTasks] = state.StatusCompleted
	st.CurrentPhase = state.PhaseApply
	st.CreatedAt = now
	st.UpdatedAt = now

	// Capture git HEAD.
	if sha, err := gitHeadSHA(projectRoot); err == nil {
		st.BaseRef = sha
	}

	statePath := filepath.Join(changeDir, "state.json")
	if err := state.Save(st, statePath); err != nil {
		return errs.WriteError(stderr, "quickstart", err)
	}

	out := struct {
		Command      string   `json:"command"`
		Status       string   `json:"status"`
		Change       string   `json:"change"`
		Description  string   `json:"description"`
		ChangeDir    string   `json:"change_dir"`
		CurrentPhase string   `json:"current_phase"`
		SpecFrom     string   `json:"spec_from"`
		Skipped      []string `json:"skipped_phases"`
	}{
		Command:      "quickstart",
		Status:       "success",
		Change:       name,
		Description:  description,
		ChangeDir:    changeDir,
		CurrentPhase: "apply",
		SpecFrom:     specPath,
		Skipped:      []string{"explore", "propose", "spec", "design", "tasks"},
	}

	writeJSON(stdout, out)
	return nil
}
