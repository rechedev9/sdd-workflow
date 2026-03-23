package context

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleReview builds context for the review phase.
// Includes: spec files, design.md, git diff of changed files, sdd-review SKILL.md.
// Optionally includes AGENTS.md / CLAUDE.md if present.
func AssembleReview(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-review"),
		loadSpecsLoader(p.ChangeDir),
		artifactLoader(p.ChangeDir, "design.md"),
		artifactLoader(p.ChangeDir, "tasks.md"),
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
	loadErr := ls.LoadAll()
	if e := checkSkillError(ls, loadErr); e != nil {
		return e
	}
	if loadErr != nil {
		if _, e := ls.Get(1); e != nil {
			return errRequiredArtifact("review", "spec artifacts", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("review", "design artifact", e)
		}
		if _, e := ls.Get(3); e != nil {
			return errRequiredArtifact("review", "tasks artifact", e)
		}
	}

	skill, _ := ls.Get(0)
	specs, _ := ls.Get(1)
	design, _ := ls.Get(2)
	tasks, _ := ls.Get(3)
	diff, _ := ls.Get(4)
	rules, _ := ls.Get(5)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	tasksStr := string(tasks)
	writeSection(w, "SPECIFICATIONS", specs)
	writeSection(w, "DESIGN", design)
	writeSectionStr(w, "COMPLETED TASKS", extractCompletedTasks(tasksStr))
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
// The two git commands are issued in parallel to halve wall-clock latency
// on repos where git diff takes tens of milliseconds.
func gitDiff(projectDir string) (string, error) {
	type result struct {
		out []byte
		err error
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()

	var unstagedRes, stagedRes result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cmd := exec.CommandContext(ctx, "git", "diff")
		cmd.Dir = projectDir
		unstagedRes.out, unstagedRes.err = cmd.Output()
	}()

	go func() {
		defer wg.Done()
		cmd := exec.CommandContext(ctx, "git", "diff", "--cached")
		cmd.Dir = projectDir
		stagedRes.out, stagedRes.err = cmd.Output()
	}()

	wg.Wait()

	if unstagedRes.err != nil {
		return "", fmt.Errorf("git diff: %w", unstagedRes.err)
	}
	if stagedRes.err != nil {
		return "", fmt.Errorf("git diff --cached: %w", stagedRes.err)
	}

	unstaged, staged := unstagedRes.out, stagedRes.out
	if len(staged) == 0 && len(unstaged) == 0 {
		return "(no changes)", nil
	}
	var buf strings.Builder
	if len(staged) > 0 {
		buf.WriteString("=== STAGED ===\n")
		buf.Write(staged)
	}
	if len(unstaged) > 0 {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("=== UNSTAGED ===\n")
		buf.Write(unstaged)
	}
	return buf.String(), nil
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
