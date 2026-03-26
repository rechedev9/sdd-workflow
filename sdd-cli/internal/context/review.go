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
// Includes: proposal.md, spec files, design.md, git diff of changed files, sdd-review SKILL.md.
// Optionally includes AGENTS.md / CLAUDE.md if present.
func AssembleReview(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		skillLoader(p.SkillsPath, "sdd-review"),
		artifactLoader(p.ChangeDir, "proposal.md"),
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
				return nil, nil //nolint:nilerr // project rules are optional
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
			return errRequiredArtifact("review", "proposal artifact", e)
		}
		if _, e := ls.Get(2); e != nil {
			return errRequiredArtifact("review", "spec artifacts", e)
		}
		if _, e := ls.Get(3); e != nil {
			return errRequiredArtifact("review", "design artifact", e)
		}
		if _, e := ls.Get(4); e != nil {
			return errRequiredArtifact("review", "tasks artifact", e)
		}
	}

	skill, _ := ls.Get(0)
	proposal, _ := ls.Get(1)
	specs, _ := ls.Get(2)
	design, _ := ls.Get(3)
	tasks, _ := ls.Get(4)
	diff, _ := ls.Get(5)
	rules, _ := ls.Get(6)

	writeSection(w, "SKILL", skill)

	writeChangeSection(w, p)

	tasksStr := string(tasks)
	if p.Compact {
		writeSectionStr(w, "PROPOSAL (compact)", compactProposal(string(proposal)))
		writeSectionStr(w, "SPECIFICATIONS (compact)", compactSpecs(string(specs)))
		writeSectionStr(w, "DESIGN (compact)", compactDesign(string(design)))
	} else {
		writeSection(w, "PROPOSAL", proposal)
		writeSection(w, "SPECIFICATIONS", specs)
		writeSection(w, "DESIGN", design)
	}
	writeSectionStr(w, "COMPLETED TASKS", extractCompletedTasks(tasksStr))
	if p.Compact {
		writeSectionStr(w, "CURRENT TASK", extractCurrentTask(tasksStr))
	} else {
		writeSection(w, "TASKS", tasks)
	}
	if len(diff) > 0 {
		writeSection(w, "GIT DIFF", diff)
	}

	if len(rules) > 0 {
		writeSection(w, "PROJECT RULES", rules)
	}

	return nil
}

// maxDiffBytes caps the combined git diff output included in review context.
// Large refactors can produce 50KB+ diffs that push the review phase past
// the 100KB size guard. 10KB is enough to review meaningful changes; for the
// full diff the reviewer can run `git diff` directly.
const maxDiffBytes = 10 * 1024

// gitDiff runs git diff and returns staged + unstaged changes.
// The two git commands are issued in parallel to halve wall-clock latency
// on repos where git diff takes tens of milliseconds.
// Output is truncated to maxDiffBytes to keep review context within budget.
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
	buf.Grow(len(staged) + len(unstaged) + 36) // pre-size: headers are ~36 bytes total
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

	// Truncate to keep review context within token budget.
	if buf.Len() > maxDiffBytes {
		s := buf.String()[:maxDiffBytes]
		return s + "\n\n... (truncated at 10KB — run `git diff` for full output)\n", nil
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
