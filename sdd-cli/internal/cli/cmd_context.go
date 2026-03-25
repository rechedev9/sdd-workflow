package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func runContext(args []string, stdout io.Writer, stderr io.Writer) error {
	args, verbosity := ParseVerbosityFlags(args)
	jsonOut := false
	compact := false
	var positional []string
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOut = true
		case arg == "--compact":
			compact = true
		case !strings.HasPrefix(arg, "-"):
			positional = append(positional, arg)
		default:
			return errUnknownFlag(arg)
		}
	}

	if len(positional) < 1 {
		return errs.Usage("usage: sdd context <name> [phase] [--json]")
	}

	name := positional[0]

	changeDir, st, err := loadChangeState(stderr, "context", name)
	if err != nil {
		return err
	}

	projectRoot, err := getProjectRoot(stderr, "context")
	if err != nil {
		return err
	}

	// Load config.
	cfg, err := loadConfig(stderr, "context", projectRoot)
	if err != nil {
		return err
	}

	db := tryOpenStore(projectRoot)
	if db != nil {
		defer db.Close()
	}
	broker := newBroker(int(verbosity), db)
	p := &sddctx.Params{
		ChangeDir:   changeDir,
		ChangeName:  st.Name,
		Description: st.Description,
		ProjectDir:  projectRoot,
		Config:      cfg,
		SkillsPath:  cfg.SkillsPath,
		Broker:      broker,
		Compact:     compact,
	}

	// Choose target writer: buffer for JSON mode, stdout otherwise.
	var target io.Writer
	var buf *bytes.Buffer
	if jsonOut {
		buf = &bytes.Buffer{}
		target = buf
	} else {
		target = stdout
	}

	// Determine phase and assemble.
	var phase string
	if len(positional) >= 2 {
		// Explicit phase arg → single assembly.
		ph, err := state.ResolvePhase(positional[1])
		if err != nil {
			return errs.WriteError(stderr, "context", err)
		}
		phase = string(ph)
		if err := sddctx.Assemble(target, ph, p); err != nil {
			return errs.WriteError(stderr, "context", err)
		}
	} else {
		// Auto-resolve: check if multiple phases are ready (spec+design parallel window).
		ready := st.ReadyPhases()
		if len(ready) == 0 {
			return errs.WriteError(stderr, "context", fmt.Errorf("no phases ready (pipeline complete or blocked)"))
		}
		if len(ready) > 1 {
			// Concurrent assembly for parallel phases (spec+design).
			names := make([]string, len(ready))
			for i, r := range ready {
				names[i] = string(r)
			}
			phase = strings.Join(names, "+")
			if err := sddctx.AssembleConcurrent(target, ready, p); err != nil {
				return errs.WriteError(stderr, "context", err)
			}
		} else {
			phase = string(ready[0])
			if err := sddctx.Assemble(target, ready[0], p); err != nil {
				return errs.WriteError(stderr, "context", err)
			}
		}
	}

	if jsonOut {
		content := buf.String()
		out := struct {
			Command string `json:"command"`
			Status  string `json:"status"`
			Change  string `json:"change"`
			Phase   string `json:"phase"`
			Context string `json:"context"`
			Bytes   int    `json:"bytes"`
			Tokens  int    `json:"tokens"`
		}{
			Command: "context",
			Status:  "success",
			Change:  name,
			Phase:   phase,
			Context: content,
			Bytes:   len(content),
			Tokens:  len(content) / 4,
		}
		writeJSON(stdout, out)
	}

	return nil
}
