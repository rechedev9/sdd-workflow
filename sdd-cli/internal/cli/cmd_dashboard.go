package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/dashboard"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/store"
)

func runDashboard(args []string, stdout io.Writer, stderr io.Writer) error {
	port := "8811"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case (arg == "--port" || arg == "-p") && i+1 < len(args):
			port = args[i+1]
			i++ // skip port value
		default:
			return errUnknownFlag(arg)
		}
	}

	p, err := strconv.Atoi(port)
	if err != nil || p < 1024 || p > 65535 {
		return errs.Usage(fmt.Sprintf("invalid port: %s (must be 1024-65535)", port))
	}

	cwd, err := getCWD(stderr, "dashboard")
	if err != nil {
		return err
	}
	dbPath := openspecDB(cwd)
	changesDir := openspecChanges(cwd)

	db, err := store.Open(dbPath)
	if err != nil {
		return errs.WriteError(stderr, "dashboard", fmt.Errorf("open store: %w", err))
	}
	defer db.Close()

	srv := dashboard.New(db, changesDir)
	addr := "0.0.0.0:" + port

	out := struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		URL     string `json:"url"`
	}{
		Command: "dashboard",
		Status:  "running",
		URL:     "http://" + addr,
	}
	writeJSON(stdout, out)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go srv.Hub().Run(ctx)

	slog.Info("dashboard started", "url", "http://"+addr)
	return srv.ListenAndServe(ctx, addr)
}
