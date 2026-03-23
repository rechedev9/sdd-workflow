package context

import (
	"io"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleApply builds context for the apply phase.
// Includes: tasks.md (current incomplete task only), design.md, spec files,
// sdd-apply SKILL.md.
func AssembleApply(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-apply"),
		artifactLoader(p.ChangeDir, "tasks.md"),
		artifactLoader(p.ChangeDir, "design.md"),
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
			return errRequiredArtifact("apply", "tasks artifact", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("apply", "design artifact", e)
		}
		if _, e := ls.Get(3); e != nil {
			return errRequiredArtifact("apply", "spec artifacts", e)
		}
	}

	skill, _ := ls.Get(0)
	tasksRaw, _ := ls.Get(1)
	design, _ := ls.Get(2)
	specs, _ := ls.Get(3)
	summary, _ := ls.Get(4)

	tasksStr := string(tasksRaw)
	currentTask := extractCurrentTask(tasksStr)
	completedSummary := extractCompletedTasks(tasksStr)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	if len(summary) > 0 {
		writeSection(w, "PIPELINE CONTEXT", summary)
	}

	writeSectionStr(w, "COMPLETED TASKS", completedSummary)
	writeSectionStr(w, "CURRENT TASK", currentTask)
	writeSection(w, "DESIGN", design)
	writeSection(w, "SPECIFICATIONS", specs)

	return nil
}

// extractCurrentTask finds the first incomplete task section in tasks.md.
// Returns the section header + all tasks in that section (both complete and incomplete).
// If no incomplete task exists, returns the full content.
func extractCurrentTask(tasks string) string {
	lines := strings.Split(tasks, "\n")
	firstIncomplete := -1

	// Find the first unchecked task.
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- [ ]") {
			firstIncomplete = i
			break
		}
	}

	if firstIncomplete == -1 {
		return tasks
	}

	// Walk back to find the section header (## or # level only).
	// Stops at ## or # to avoid splitting a section at a sub-header (###).
	start := firstIncomplete
	for j := firstIncomplete - 1; j >= 0; j-- {
		h := strings.TrimSpace(lines[j])
		if strings.HasPrefix(h, "## ") || h == "##" ||
			strings.HasPrefix(h, "# ") || h == "#" {
			start = j
			break
		}
	}

	// Walk forward to find the next section header (## level only).
	// Stops at ## or # to avoid splitting a section at a sub-header (###).
	end := len(lines)
	for i := firstIncomplete + 1; i < len(lines); i++ {
		h := strings.TrimSpace(lines[i])
		if strings.HasPrefix(h, "## ") || h == "##" ||
			strings.HasPrefix(h, "# ") || h == "#" {
			end = i
			break
		}
	}

	return strings.Join(lines[start:end], "\n")
}
