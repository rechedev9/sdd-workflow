package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/artifacts"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runDump(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd dump <name>")
	}

	name := args[0]
	if len(args) > 1 {
		return errUnknownFlag(args[1])
	}

	changeDir, st, err := loadChangeState(stderr, "dump", name)
	if err != nil {
		return err
	}

	projectRoot, err := getProjectRoot(stderr, "dump")
	if err != nil {
		return err
	}

	// Load config.
	cfg, err := loadConfig(stderr, "dump", projectRoot)
	if err != nil {
		return err
	}

	// List artifacts.
	arts, err := artifacts.List(changeDir)
	if err != nil {
		return errs.WriteError(stderr, "dump", fmt.Errorf("list artifacts: %w", err))
	}

	pending, err := artifacts.ListPending(changeDir)
	if err != nil {
		return errs.WriteError(stderr, "dump", fmt.Errorf("list pending: %w", err))
	}
	if pending == nil {
		pending = []artifacts.ArtifactInfo{}
	}

	// Load pipeline metrics.
	pm := sddctx.LoadPipelineMetrics(changeDir)

	// Read cache hash files.
	cacheDir := filepath.Join(changeDir, ".cache")
	hashFiles, _ := filepath.Glob(filepath.Join(cacheDir, "*.hash"))
	cacheKeys := make(map[string]string, len(hashFiles))
	for _, hf := range hashFiles {
		base := strings.TrimSuffix(filepath.Base(hf), ".hash")
		raw, err := os.ReadFile(hf)
		if err != nil {
			continue
		}
		trimmed := bytes.TrimSpace(raw)
		// Hash files use "{hash}|{unix_seconds}" format; expose only the hash portion.
		if hashPart, _, found := bytes.Cut(trimmed, []byte("|")); found {
			cacheKeys[base] = string(hashPart)
		} else {
			cacheKeys[base] = string(trimmed)
		}
	}

	out := struct {
		Command   string                   `json:"command"`
		Status    string                   `json:"status"`
		Change    string                   `json:"change"`
		State     *state.State             `json:"state"`
		Config    *config.Config           `json:"config"`
		Artifacts []artifacts.ArtifactInfo `json:"artifacts"`
		Pending   []artifacts.ArtifactInfo `json:"pending"`
		Metrics   *sddctx.PipelineMetrics  `json:"metrics"`
		CacheKeys map[string]string        `json:"cache_keys"`
	}{
		Command:   "dump",
		Status:    "success",
		Change:    name,
		State:     st,
		Config:    cfg,
		Artifacts: arts,
		Pending:   pending,
		Metrics:   pm,
		CacheKeys: cacheKeys,
	}

	writeJSON(stdout, out)
	return nil
}
