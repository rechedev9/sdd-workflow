package context

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleDesign builds context for the design phase.
// Includes: spec files (MUST/SHOULD requirements), proposal.md, sdd-design SKILL.md.
func AssembleDesign(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-design"),
		artifactLoader(p.ChangeDir, "proposal.md"),
		loadSpecsLoader(p.ChangeDir),
		buildSummaryLoader(p),
	}

	ls := csync.NewLazySlice(loaders)
	loadErr := ls.LoadAll()
	if e := checkSkillError(ls, loadErr); e != nil {
		return e
	}
	if loadErr != nil {
		if _, e := ls.Get(1); e != nil {
			return errRequiredArtifact("design", "proposal artifact", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("design", "spec artifacts", e)
		}
	}

	skill, _ := ls.Get(0)
	proposal, _ := ls.Get(1)
	specs, _ := ls.Get(2)
	summary, _ := ls.Get(3)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	writeSectionStr(w, "PROJECT", projectContext(p))

	if len(summary) > 0 {
		writeSection(w, "PIPELINE CONTEXT", summary)
	}

	writeSection(w, "PROPOSAL", proposal)
	writeSection(w, "SPECIFICATIONS", specs)

	return nil
}

// artifactLoader returns a loader closure that reads a named artifact file.
// Used by assemblers to register artifact loads as lazy-load tasks.
func artifactLoader(changeDir, name string) func() ([]byte, error) {
	return func() ([]byte, error) { return loadArtifact(changeDir, name) }
}

// skillLoader returns a loader closure that reads a named skill file.
// Used by assemblers to register skill loads as lazy-load tasks.
func skillLoader(skillsPath, name string) func() ([]byte, error) {
	return func() ([]byte, error) { return loadSkill(skillsPath, name) }
}

// loadSpecsLoader returns a loader closure that reads spec files as bytes.
// Used by assemblers that register loadSpecs as a lazy-load task.
func loadSpecsLoader(changeDir string) func() ([]byte, error) {
	return func() ([]byte, error) {
		s, err := loadSpecs(changeDir)
		return []byte(s), err
	}
}

// buildSummaryLoader returns a loader closure that builds the pipeline summary.
// Used by assemblers that include a cumulative context section.
func buildSummaryLoader(p *Params) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(buildSummary(p.ChangeDir, p)), nil }
}

// loadSpecs reads all .md files from the specs/ directory, concatenated.
func loadSpecs(changeDir string) (string, error) {
	specsDir := filepath.Join(changeDir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return "", fmt.Errorf("read specs directory: %w", err)
	}

	var b strings.Builder
	// Pre-size: sum .md file sizes (from DirEntry.Info, cached from ReadDir) + headers.
	// Filters match the load loop to avoid over-allocating for dirs and non-.md files.
	var totalEst int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if info, err := e.Info(); err == nil {
			totalEst += int(info.Size()) + 20
		}
	}
	if totalEst > 0 {
		b.Grow(totalEst)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(specsDir, e.Name()))
		if err != nil {
			continue
		}
		if count > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("### ")
		b.WriteString(e.Name())
		b.WriteString("\n\n")
		b.Write(data)
		count++
	}

	if count == 0 {
		return "", fmt.Errorf("no spec files found in %s", specsDir)
	}

	return b.String(), nil
}
