package verify

import (
	"fmt"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/fsutil"
)

// ShipResult holds the outcome of a ship operation.
type ShipResult struct {
	Branch  string   `json:"branch"`
	PRURL   string   `json:"pr_url"`
	Files   []string `json:"files"`
	BaseSHA string   `json:"base_sha"`
}

// WriteShipReport writes ship-report.md to changeDir.
func WriteShipReport(result *ShipResult, changeDir string) error {
	var b strings.Builder

	fmt.Fprintf(&b, "# Ship Report\n\n")
	fmt.Fprintf(&b, "**Shipped:** %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "**Branch:** `%s`\n\n", result.Branch)
	fmt.Fprintf(&b, "**PR:** %s\n\n", result.PRURL)
	fmt.Fprintf(&b, "**Base SHA:** `%s`\n\n", result.BaseSHA)
	fmt.Fprintf(&b, "## Files (%d)\n\n", len(result.Files))
	for _, f := range result.Files {
		fmt.Fprintf(&b, "- %s\n", f)
	}

	return fsutil.AtomicWrite(changeDir+"/ship-report.md", []byte(b.String()))
}
