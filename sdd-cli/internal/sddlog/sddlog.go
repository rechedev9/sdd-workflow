// Package sddlog configures structured logging for sdd-cli.
//
// By default, logs go to stderr as human-readable text.
// Set SDD_LOG=json for machine-parseable JSON output.
// Set SDD_LOG_FILE=<path> to also write logs to a file.
package sddlog

import (
	"io"
	"log/slog"
	"os"
)

// Init configures the global slog logger based on environment variables.
// Returns a cleanup function that should be deferred by the caller.
func Init(stderr io.Writer) func() {
	var writers []io.Writer
	writers = append(writers, stderr)

	var cleanup func()
	if path := os.Getenv("SDD_LOG_FILE"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			writers = append(writers, f)
			cleanup = func() { f.Close() }
		}
	}

	w := io.MultiWriter(writers...)

	var handler slog.Handler
	if os.Getenv("SDD_LOG") == "json" {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(handler))

	if cleanup == nil {
		return func() {}
	}
	return cleanup
}
