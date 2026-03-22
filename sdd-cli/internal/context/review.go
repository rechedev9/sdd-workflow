package context

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleReview builds context for the review phase.
// Includes: spec files, design.md, git diff of changed files, sdd-review SKILL.md.
// Optionally includes AGENTS.md / CLAUDE.md if present.
func AssembleReview(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-review") },
		func() ([]byte, error) {
			s, err := loadSpecs(p.ChangeDir)
			return []byte(s), err
		},
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "design.md") },
		func() ([]byte, error) { return loadArtifact(p.ChangeDir, "tasks.md") },
		func() ([]byte, error) {
			d, err := gitDiff(p.ProjectDir)
			if err != nil {
				return []byte(fmt.Sprintf("(git diff unavailable: %v)", err)), nil
			}
			return []byte(d), nil
		},
		func() ([]byte, error) {
			rules, err := loadProjectRules(p.ProjectDir)
			if err != nil {
				return nil, nil // non-fatal
			}
			return rules, nil
		},
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e
		}
		if _, e := ls.Get(1); e != nil {
			return fmt.Errorf("review requires spec artifacts: %w", e)
		}
		if _, e := ls.Get(2); e != nil {
			return fmt.Errorf("review requires design artifact: %w", e)
		}
		if _, e := ls.Get(3); e != nil {
			return fmt.Errorf("review requires tasks artifact: %w", e)
		}
	}

	skill, _ := ls.Get(0)
	specs, _ := ls.Get(1)
	design, _ := ls.Get(2)
	tasks, _ := ls.Get(3)
	diff, _ := ls.Get(4)
	rules, _ := ls.Get(5)

	writeSection(w, "SKILL", skill)

	writeSectionStr(w, "CHANGE", fmt.Sprintf(
		"Name: %s\nDescription: %s",
		p.ChangeName, p.Description,
	))

	writeSection(w, "SPECIFICATIONS", specs)
	writeSection(w, "DESIGN", design)
	writeSectionStr(w, "COMPLETED TASKS", extractCompletedTasks(string(tasks)))
	writeSection(w, "TASKS", tasks)
	if len(diff) > 0 {
		writeSection(w, "GIT DIFF", diff)
	}

	if len(rules) > 0 {
		writeSection(w, "PROJECT RULES", rules)
	}

	return nil
}

// gitDiff runs git diff and returns staged + unstaged changes.
func gitDiff(projectDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	// Unstaged changes.
	cmd := exec.CommandContext(ctx, "git", "diff")
	cmd.Dir = projectDir
	unstaged, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}

	// Staged changes.
	cmd = exec.CommandContext(ctx, "git", "diff", "--cached")
	cmd.Dir = projectDir
	staged, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff --cached: %w", err)
	}

	var parts []string
	if len(staged) > 0 {
		parts = append(parts, "=== STAGED ===\n"+string(staged))
	}
	if len(unstaged) > 0 {
		parts = append(parts, "=== UNSTAGED ===\n"+string(unstaged))
	}
	if len(parts) == 0 {
		return "(no changes)", nil
	}
	return strings.Join(parts, "\n"), nil
}

// loadProjectRules tries to load AGENTS.md or CLAUDE.md from the project root.
func loadProjectRules(projectDir string) ([]byte, error) {
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := loadArtifact(projectDir, name)
		if err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("no project rules file found")
}
