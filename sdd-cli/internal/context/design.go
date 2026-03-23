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
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-design") },
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "proposal.md") },
		func() ([]byte, error) {
			s, err := loadSpecs(p.ChangeDir)
			return []byte(s), err
		},
		func() ([]byte, error) { return []byte(buildSummary(p.ChangeDir, p)), nil },
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return fmt.Errorf("design requires proposal artifact: %w", e)
		}
		if _, e := ls.Get(2); e != nil {
			return fmt.Errorf("design requires spec artifacts: %w", e)
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

// loadSpecs reads all .md files from the specs/ directory, concatenated.
func loadSpecs(changeDir string) (string, error) {
	specsDir := filepath.Join(changeDir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return "", fmt.Errorf("read specs directory: %w", err)
	}

	var b strings.Builder
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
