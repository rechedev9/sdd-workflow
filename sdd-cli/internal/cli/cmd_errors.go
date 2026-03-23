package cli

import (
	"cmp"
	"fmt"
	"io"
	"slices"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
)

func runErrors(args []string, stdout io.Writer, stderr io.Writer) error {
	jsonOut := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOut = true
		default:
			return errUnknownFlag(arg)
		}
	}

	cwd, err := getCWD(stderr, "errors")
	if err != nil {
		return err
	}

	log := errlog.Load(cwd)

	if jsonOut {
		type errorGroup struct {
			Fingerprint string   `json:"fingerprint"`
			Count       int      `json:"count"`
			Command     string   `json:"command"`
			LastSeen    string   `json:"last_seen"`
			ErrorLines  []string `json:"error_lines"`
		}
		groups := make(map[string]*errorGroup, len(log.Entries))
		for _, e := range log.Entries {
			lines := e.ErrorLines
			if lines == nil {
				lines = []string{}
			}
			g, ok := groups[e.Fingerprint]
			if !ok {
				g = &errorGroup{
					Fingerprint: e.Fingerprint,
					Command:     e.Command,
					ErrorLines:  lines,
				}
				groups[e.Fingerprint] = g
			}
			g.Count++
			if e.Timestamp > g.LastSeen {
				g.LastSeen = e.Timestamp
				g.ErrorLines = lines
			}
		}

		sorted := make([]*errorGroup, 0, len(groups))
		for _, g := range groups {
			sorted = append(sorted, g)
		}
		slices.SortFunc(sorted, func(a, b *errorGroup) int {
			return cmp.Compare(b.Count, a.Count)
		})

		out := struct {
			Command string        `json:"command"`
			Status  string        `json:"status"`
			Total   int           `json:"total"`
			Groups  []*errorGroup `json:"groups"`
		}{
			Command: "errors",
			Status:  "success",
			Total:   len(log.Entries),
			Groups:  sorted,
		}
		writeJSON(stdout, out)
		return nil
	}

	if len(log.Entries) == 0 {
		fmt.Fprintln(stdout, "sdd errors: no recorded errors")
		return nil
	}

	counts := log.RecurringFingerprints(1)
	fmt.Fprintf(stdout, "sdd errors: %d entries, %d unique patterns\n\n", len(log.Entries), len(counts))
	for _, e := range log.Entries[max(0, len(log.Entries)-10):] {
		fmt.Fprintf(stdout, "  %.19s  %-8s  exit=%d  %s  [%.8s]\n",
			e.Timestamp, e.CommandName, e.ExitCode, e.Change, e.Fingerprint)
	}
	return nil
}
