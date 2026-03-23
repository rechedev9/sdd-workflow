package context

import (
	"io"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleClean builds context for the clean phase.
// Includes: verify-report.md, tasks.md, design.md, specs, cumulative summary,
// sdd-clean SKILL.md.
func AssembleClean(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-clean"),
		artifactLoader(p.ChangeDir, "verify-report.md"),
		artifactLoader(p.ChangeDir, "tasks.md"),
		buildSummaryLoader(p),
		artifactLoader(p.ChangeDir, "design.md"),
		func() ([]byte, error) {
			s, err := loadSpecs(p.ChangeDir)
			if err != nil {
				return nil, nil // non-fatal
			}
			return []byte(s), nil
		},
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return errRequiredArtifact("clean", "verify-report artifact", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("clean", "tasks artifact", e)
		}
	}

	skill, _ := ls.Get(0)
	verifyReport, _ := ls.Get(1)
	tasks, _ := ls.Get(2)
	summary, _ := ls.Get(3)
	design, _ := ls.Get(4)
	specs, _ := ls.Get(5)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	if len(summary) > 0 {
		writeSection(w, "PIPELINE CONTEXT", summary)
	}

	tasksStr := string(tasks)
	writeSection(w, "VERIFY REPORT", verifyReport)
	writeSectionStr(w, "COMPLETED TASKS", extractCompletedTasks(tasksStr))
	writeSection(w, "TASKS", tasks)

	if len(design) > 0 {
		writeSection(w, "DESIGN", design)
	}
	if len(specs) > 0 {
		writeSection(w, "SPECIFICATIONS", specs)
	}

	return nil
}
