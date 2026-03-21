package context

import (
	"fmt"
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleTasks builds context for the tasks phase.
// Includes: spec files, design.md, sdd-tasks SKILL.md.
func AssembleTasks(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-tasks") },
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "design.md") },
		func() ([]byte, error) {
			s, err := loadSpecs(p.ChangeDir)
			return []byte(s), err
		},
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return fmt.Errorf("tasks requires design artifact: %w", e)
		}
		if _, e := ls.Get(2); e != nil {
			return fmt.Errorf("tasks requires spec artifacts: %w", e)
		}
	}

	skill, _ := ls.Get(0)
	design, _ := ls.Get(1)
	specs, _ := ls.Get(2)

	writeSection(w, "SKILL", skill)

	writeSectionStr(w, "CHANGE", fmt.Sprintf(
		"Name: %s\nDescription: %s",
		p.ChangeName, p.Description,
	))

	writeSection(w, "SPECIFICATIONS", specs)
	writeSection(w, "DESIGN", design)

	return nil
}
