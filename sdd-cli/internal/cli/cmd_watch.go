package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/events"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/watch"
)

// runWatch implements the "sdd watch <name>" command.
// Long-running: blocks until SIGINT/SIGTERM.
func runWatch(args []string, stdout io.Writer, stderr io.Writer) error {
	args, verbosity := ParseVerbosityFlags(args)

	debounceMs := 300
	var name string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--debounce":
			if i+1 >= len(args) {
				return errs.Usage("--debounce requires a value in milliseconds")
			}
			i++
			val, err := strconv.Atoi(args[i])
			if err != nil {
				return errs.Usage(fmt.Sprintf("invalid debounce value: %s (must be a positive integer)", args[i]))
			}
			if val <= 0 {
				return errs.Usage(fmt.Sprintf("invalid debounce value: %d (must be > 0)", val))
			}
			debounceMs = val
		case strings.HasPrefix(arg, "-"):
			return errUnknownFlag(arg)
		default:
			if name != "" {
				return errs.Usage("watch accepts exactly one positional argument: <name>")
			}
			name = arg
		}
	}

	if name == "" {
		return errs.Usage("usage: sdd watch <name> [--debounce <ms>]")
	}

	changeDir, st, err := loadChangeState(stderr, "watch", name)
	if err != nil {
		return err
	}

	projectRoot, err := getProjectRoot(stderr, "watch")
	if err != nil {
		return err
	}

	cfg, err := loadConfig(stderr, "watch", projectRoot)
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
	}

	// Emit JSON startup message.
	out := struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		Change  string `json:"change"`
		Phase   string `json:"phase"`
		Dir     string `json:"dir"`
	}{
		Command: "watch",
		Status:  "watching",
		Change:  name,
		Phase:   string(st.CurrentPhase),
		Dir:     changeDir,
	}
	writeJSON(stdout, out)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reassemble := func(ctx context.Context, w io.Writer) error {
		// Re-read state on every trigger (phase may have advanced).
		st, err := state.Load(filepath.Join(changeDir, "state.json"))
		if err != nil {
			return fmt.Errorf("reload state: %w", err)
		}

		ready := st.ReadyPhases()
		if len(ready) == 0 {
			slog.Info("watch: no phases ready (pipeline complete or blocked)")
			return nil
		}

		start := time.Now()

		// Update params for current state.
		p.ChangeName = st.Name
		p.Description = st.Description

		var phase string
		if len(ready) > 1 {
			names := make([]string, len(ready))
			for i, r := range ready {
				names[i] = string(r)
			}
			phase = strings.Join(names, "+")
			if err := sddctx.AssembleConcurrent(w, ready, p); err != nil {
				return err
			}
		} else {
			phase = string(ready[0])
			if err := sddctx.Assemble(w, ready[0], p); err != nil {
				return err
			}
		}

		broker.Emit(events.Event{
			Type: events.WatchReassembled,
			Payload: events.WatchReassembledPayload{
				Change:     name,
				Phase:      phase,
				DurationMs: time.Since(start).Milliseconds(),
			},
		})

		return nil
	}

	return watch.Run(ctx, watch.Options{
		ChangeDir:  changeDir,
		Debounce:   time.Duration(debounceMs) * time.Millisecond,
		Stdout:     stdout,
		Stderr:     stderr,
		Reassemble: reassemble,
		Broker:     broker,
		ChangeName: name,
	})
}
