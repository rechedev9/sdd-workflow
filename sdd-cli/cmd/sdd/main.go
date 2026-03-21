package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli"
)

// startProfile starts CPU and/or heap profiling based on mode.
// Returns a cleanup function that stops profiling and closes files.
// mode: "cpu", "mem", "all", or "" (no-op).
func startProfile(mode string, stderr *os.File) func() {
	if mode == "" {
		return func() {}
	}

	var closers []func()

	if mode == "cpu" || mode == "all" {
		f, err := os.Create("sdd-cpu.prof")
		if err != nil {
			fmt.Fprintf(stderr, "sdd: cannot create cpu profile: %v\n", err)
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				fmt.Fprintf(stderr, "sdd: cannot start cpu profile: %v\n", err)
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
				fmt.Fprintf(stderr, "sdd: cannot create mem profile: %v\n", err)
				return
			}
			defer f.Close()
			runtime.GC()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(stderr, "sdd: cannot write mem profile: %v\n", err)
			}
		})
	}

	if mode != "cpu" && mode != "mem" && mode != "all" {
		fmt.Fprintf(stderr, "sdd: unknown SDD_PPROF value %q (use cpu, mem, or all)\n", mode)
		return func() {}
	}

	return func() {
		for _, c := range closers {
			c()
		}
	}
}

func main() {
	stopProfile := startProfile(os.Getenv("SDD_PPROF"), os.Stderr)
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
			if err := os.WriteFile(name, []byte(content), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "sdd: panic recovered; failed to write crash log: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "sdd: panic recovered; crash log written to %s\n", name)
			}
			os.Exit(3)
		}
	}()

	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(cli.ExitCode(err))
	}
}
