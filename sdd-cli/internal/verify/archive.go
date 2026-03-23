package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/fsutil"
)

// ArchiveResult holds the outcome of an archive operation.
type ArchiveResult struct {
	ArchivePath  string `json:"archive_path"`
	ManifestPath string `json:"manifest_path"`
}

// Archive moves changeDir into openspec/changes/archive/{timestamp}-{name}/.
func Archive(changeDir string) (*ArchiveResult, error) {
	name := filepath.Base(changeDir)
	changesDir := filepath.Dir(changeDir)
	archiveParent := filepath.Join(changesDir, "archive")

	if err := os.MkdirAll(archiveParent, 0o755); err != nil {
		return nil, fmt.Errorf("create archive directory: %w", err)
	}

	stamp := time.Now().UTC().Format("2006-01-02-150405")
	archiveName := stamp + "-" + name
	archivePath := filepath.Join(archiveParent, archiveName)

	if err := os.Rename(changeDir, archivePath); err != nil {
		return nil, fmt.Errorf("move change to archive: %w", err)
	}

	manifestPath := filepath.Join(archivePath, "archive-manifest.md")
	if err := writeManifest(archivePath, name, manifestPath); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return &ArchiveResult{
		ArchivePath:  archivePath,
		ManifestPath: manifestPath,
	}, nil
}

// writeManifest creates archive-manifest.md listing all archived artifacts.
func writeManifest(archivePath, changeName, manifestPath string) error {
	entries, err := os.ReadDir(archivePath)
	if err != nil {
		return fmt.Errorf("read archive directory: %w", err)
	}

	var b strings.Builder
	b.Grow(200 + len(entries)*20) // pre-size: header + ~20 bytes per entry
	b.WriteString("# Archive Manifest\n\n")
	fmt.Fprintf(&b, "**Change:** %s\n", changeName)
	fmt.Fprintf(&b, "**Archived:** %s\n\n", time.Now().UTC().Format(time.RFC3339))

	b.WriteString("## Artifacts\n\n")

	specCount := 0
	completed := 0
	for _, e := range entries {
		name := e.Name()
		if name == "archive-manifest.md" || (e.IsDir() && name == ".pending") {
			continue
		}
		if e.IsDir() && name == "specs" {
			specEntries, _ := os.ReadDir(filepath.Join(archivePath, "specs"))
			specCount = len(specEntries)
			fmt.Fprintf(&b, "- `specs/` (%d files)\n", specCount)
			continue
		}
		fmt.Fprintf(&b, "- `%s`\n", name)
		switch name {
		case "exploration.md", "proposal.md", "design.md", "tasks.md",
			"review-report.md", "verify-report.md", "clean-report.md":
			completed++
		}
	}
	if specCount > 0 {
		completed++ // spec phase
	}

	// Summary section.
	b.WriteString("\n## Summary\n\n")
	fmt.Fprintf(&b, "- **Completed phases:** %d\n", completed)
	fmt.Fprintf(&b, "- **Spec files:** %d\n", specCount)

	return fsutil.AtomicWrite(manifestPath, []byte(b.String()))
}
