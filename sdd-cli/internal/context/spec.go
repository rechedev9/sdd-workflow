package context

import (
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleSpec builds context for the spec phase.
// Includes: proposal.md, cumulative summary, project stack, sdd-spec SKILL.md.
func AssembleSpec(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-spec"),
		artifactLoader(p.ChangeDir, "proposal.md"),
		buildSummaryLoader(p),
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return errRequiredArtifact("spec", "proposal artifact", e)
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
