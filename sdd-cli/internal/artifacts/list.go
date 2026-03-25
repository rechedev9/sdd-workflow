package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// ArtifactInfo describes a single artifact on disk.
type ArtifactInfo struct {
	Phase    state.Phase `json:"phase"`
	Filename string      `json:"filename"`
	Path     string      `json:"path"`
	Size     int64       `json:"size"`
}

// List returns all existing artifacts in the change directory.
func List(changeDir string) ([]ArtifactInfo, error) {
	phases := state.AllPhases()
	result := make([]ArtifactInfo, 0, len(phases))

	for _, phase := range phases {
		name, ok := ArtifactFileName(phase)
		if !ok {
			continue
		}

		path := filepath.Join(changeDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// For directory-backed phases, recurse to find all regular files.
			files, err := collectRegularFiles(path)
			if err != nil || len(files) == 0 {
				continue
			}
			for _, f := range files {
				fInfo, err := os.Stat(f)
				if err != nil {
					continue
				}
				rel, _ := filepath.Rel(changeDir, f)
				result = append(result, ArtifactInfo{
					Phase:    phase,
					Filename: rel,
					Path:     f,
					Size:     fInfo.Size(),
				})
			}
		} else {
			result = append(result, ArtifactInfo{
				Phase:    phase,
				Filename: name,
				Path:     path,
				Size:     info.Size(),
			})
		}
	}

	return result, nil
}

// ListPending returns pending artifacts in the .pending/ directory.
func ListPending(changeDir string) ([]ArtifactInfo, error) {
	pendingDir := filepath.Join(changeDir, ".pending")
	info, err := os.Stat(pendingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat .pending directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("read .pending directory: not a directory")
	}

	result := make([]ArtifactInfo, 0)
	err = filepath.Walk(pendingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(pendingDir, path)
		if err != nil {
			return fmt.Errorf("relative pending path: %w", err)
		}
		phase := pendingPhase(rel)
		result = append(result, ArtifactInfo{
			Phase:    phase,
			Filename: rel,
			Path:     path,
			Size:     info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk .pending directory: %w", err)
	}
	return result, nil
}

func pendingPhase(rel string) state.Phase {
	base := filepath.Base(rel)
	phase := state.Phase(strings.TrimSuffix(base, ".md"))
	if _, ok := ArtifactFileName(phase); ok && !isDirectoryArtifact(phase) {
		return phase
	}

	first := rel
	if idx := strings.IndexRune(rel, filepath.Separator); idx >= 0 {
		first = rel[:idx]
	}
	for _, phase := range state.AllPhases() {
		name, ok := ArtifactFileName(phase)
		if ok && isDirectoryArtifact(phase) && name == first {
			return phase
		}
	}
	return ""
}
