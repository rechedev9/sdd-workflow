package context

import (
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleTasks builds context for the tasks phase.
// Includes: spec files, design.md, sdd-tasks SKILL.md.
func AssembleTasks(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-tasks"),
		artifactLoader(p.ChangeDir, "design.md"),
		loadSpecsLoader(p.ChangeDir),
	}

	ls := csync.NewLazySlice(loaders)
	loadErr := ls.LoadAll()
	if e := checkSkillError(ls, loadErr); e != nil {
		return e
	}
	if loadErr != nil {
		if _, e := ls.Get(1); e != nil {
			return errRequiredArtifact("tasks", "design artifact", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("tasks", "spec artifacts", e)
		}
	}

	skill, _ := ls.Get(0)
	design, _ := ls.Get(1)
	specs, _ := ls.Get(2)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	writeSection(w, "SPECIFICATIONS", specs)
	writeSection(w, "DESIGN", design)

	return nil
}
