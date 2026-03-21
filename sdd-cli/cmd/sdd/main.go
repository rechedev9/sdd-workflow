package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/sddlog"
)

// startProfile starts CPU and/or heap profiling based on mode.
// Returns a cleanup function that stops profiling and closes files.
// mode: "cpu", "mem", "all", or "" (no-op).
func startProfile(mode string) func() {
	if mode == "" {
		return func() {}
	}

	var closers []func()

	if mode == "cpu" || mode == "all" {
		f, err := os.Create("sdd-cpu.prof")
		if err != nil {
			slog.Error("cannot create cpu profile", "error", err)
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				slog.Error("cannot start cpu profile", "error", err)
				f.Close()
			} else {
				closers = append(closers, func() {
					pprof.StopCPUProfile()
					f.Close()
				})
			}
		}
	}

	if mode == "mem" || mode == "all" {
		closers = append(closers, func() {
			f, err := os.Create("sdd-mem.prof")
			if err != nil {
				slog.Error("cannot create mem profile", "error", err)
				return
			}
			defer f.Close()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				slog.Error("cannot write mem profile", "error", err)
			}
		})
	}

	if mode != "cpu" && mode != "mem" && mode != "all" {
		slog.Warn("unknown SDD_PPROF value", "value", mode)
		return func() {}
	}

	return func() {
		for _, c := range closers {
			c()
		}
	}
}

func main() {
	closeLog := sddlog.Init(os.Stderr)
	defer closeLog()

	stopProfile := startProfile(os.Getenv("SDD_PPROF"))
	defer stopProfile()

	defer func() {
		if r := recover(); r != nil {
			ts := time.Now()
			name := fmt.Sprintf(".sdd-crash-%d.log", ts.Unix())
			content := fmt.Sprintf("sdd crash report\ntimestamp: %s\nargs: %s\npanic: %v\n\nstack trace:\n%s",
				ts.Format(time.RFC3339),
				strings.Join(os.Args, " "),
				r,
				debug.Stack(),
			)
			if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
				slog.Error("panic recovered; failed to write crash log", "error", err)
			} else {
				slog.Error("panic recovered", "crash_log", name)
			}
			os.Exit(3)
		}
	}()

	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
