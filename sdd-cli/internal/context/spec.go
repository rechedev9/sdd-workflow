package context

import (
	"fmt"
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleSpec builds context for the spec phase.
// Includes: proposal.md, cumulative summary, project stack, sdd-spec SKILL.md.
func AssembleSpec(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-spec") },
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "proposal.md") },
		func() ([]byte, error) { return []byte(buildSummary(p.ChangeDir, p)), nil },
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return fmt.Errorf("spec requires proposal artifact: %w", e)
		}
	}

	skill, _ := ls.Get(0)
	proposal, _ := ls.Get(1)
	summary, _ := ls.Get(2)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	writeSectionStr(w, "PROJECT", projectContext(p))

	if len(summary) > 0 {
		writeSection(w, "PIPELINE CONTEXT", summary)
	}

	writeSection(w, "PROPOSAL", proposal)

	return nil
}
