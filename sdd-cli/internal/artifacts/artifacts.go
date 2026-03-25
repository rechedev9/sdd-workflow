package artifacts

import (
	"io/fs"
	"path/filepath"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/phase"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// ArtifactFileName returns the canonical artifact filename for a phase.
func ArtifactFileName(ph state.Phase) (string, bool) {
	desc, ok := phase.DefaultRegistry.Get(string(ph))
	if !ok {
		return "", false
	}
	return desc.ArtifactFile, true
}

func isDirectoryArtifact(ph state.Phase) bool {
	name, ok := ArtifactFileName(ph)
	if !ok {
		return false
	}
	return filepath.Ext(name) == ""
}

func pendingRelativePath(ph state.Phase) string {
	if isDirectoryArtifact(ph) {
		name, _ := ArtifactFileName(ph)
		return name
	}
	return PendingFileName(ph)
}

func collectRegularFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// PendingFileName returns the filename used in .pending/ for a phase.
func PendingFileName(phase state.Phase) string {
	return string(phase) + ".md"
}
