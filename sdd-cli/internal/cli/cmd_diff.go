package cli

import (
	"fmt"
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/cli/errs"
)

func runDiff(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 1 {
		return errs.Usage("usage: sdd diff <name>")
	}

	name := args[0]
	if len(args) > 1 {
		return errUnknownFlag(args[1])
	}

	_, st, err := loadChangeState(stderr, "diff", name)
	if err != nil {
		return err
	}

	if st.BaseRef == "" {
		return errs.WriteError(stderr, "diff", fmt.Errorf("base_ref not recorded; change was created before diff support"))
	}

	projectRoot, err := getProjectRoot(stderr, "diff")
	if err != nil {
		return err
	}

	files, err := gitDiffFiles(projectRoot, st.BaseRef)
	if err != nil {
		return errs.WriteError(stderr, "diff", fmt.Errorf("git diff: %w", err))
	}
	if files == nil {
		files = []string{}
	}

	out := struct {
		Command string   `json:"command"`
		Status  string   `json:"status"`
		Change  string   `json:"change"`
		BaseRef string   `json:"base_ref"`
		Files   []string `json:"files"`
		Count   int      `json:"count"`
	}{
		Command: "diff",
		Status:  "success",
		Change:  name,
		BaseRef: st.BaseRef,
		Files:   files,
		Count:   len(files),
	}

	writeJSON(stdout, out)
	return nil
}
