package context

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/csync"
)

// AssembleExplore builds context for the explore phase.
// Includes: file tree (via git ls-files), config summary, sdd-explore SKILL.md.
func AssembleExplore(w io.Writer, p *Params) error {
	loaders := []func() ([]byte, error){
		func() ([]byte, error) { return loadSkill(p.SkillsPath, "sdd-explore") },
		func() ([]byte, error) {
			ft, err := gitFileTree(p.ProjectDir)
			return []byte(ft), err
		},
		func() ([]byte, error) {
			return []byte(loadManifestContents(p.ProjectDir, p.Config.Stack.Manifests)), nil
		},
	}

	ls := csync.NewLazySlice(loaders)
	if err := ls.LoadAll(); err != nil {
		if _, e := ls.Get(0); e != nil {
			return e // skill is critical
		}
	}

	skill, _ := ls.Get(0)
	fileTreeData, ftErr := ls.Get(1)
	manifests, _ := ls.Get(2)

	fileTree := string(fileTreeData)
	if ftErr != nil {
		fileTree = fmt.Sprintf("(git ls-files unavailable: %v)", ftErr)
	}

	writeSection(w, "SKILL", skill)

	writeSectionStr(w, "PROJECT", fmt.Sprintf(
		"Name: %s\nLanguage: %s\nBuild Tool: %s\nManifests: %s",
		p.Config.ProjectName,
		p.Config.Stack.Language,
		p.Config.Stack.BuildTool,
		strings.Join(p.Config.Stack.Manifests, ", "),
	))

	if p.Description != "" {
		writeSectionStr(w, "CHANGE", fmt.Sprintf(
			"Name: %s\nDescription: %s",
			p.ChangeName, p.Description,
		))
	}

	writeSectionStr(w, "FILE TREE", fileTree)

	if len(manifests) > 0 {
		writeSection(w, "MANIFESTS", manifests)
	}

	return nil
}

// gitCmdTimeout is the maximum time allowed for a git subprocess.
const gitCmdTimeout = 30 * time.Second

// gitFileTree runs git ls-files and returns the output.
func gitFileTree(projectDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-files: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
