package context

import (
	"fmt"
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssemblePropose builds context for the propose phase.
// Includes: exploration.md, project context, file tree, sdd-propose SKILL.md.
func AssemblePropose(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-propose") },
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "exploration.md") },
		func() ([]byte, error) {
			ft, err := gitFileTree(p.ProjectDir)
			return []byte(ft), err
		},
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return fmt.Errorf("propose requires exploration artifact: %w", e)
		}
	}

	skill, _ := ls.Get(0)
	exploration, _ := ls.Get(1)
	fileTreeData, ftErr := ls.Get(2)

	writeSection(w, "SKILL", skill)

	writeSectionStr(w, "CHANGE", fmt.Sprintf(
		"Name: %s\nDescription: %s",
		p.ChangeName, p.Description,
	))

	writeSectionStr(w, "PROJECT", projectContext(p))

	if ftErr == nil {
		writeSectionStr(w, "FILE TREE", string(fileTreeData))
	}

	writeSection(w, "EXPLORATION", exploration)

	return nil
}
