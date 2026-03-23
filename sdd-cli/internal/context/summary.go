package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// appendArtifactSection reads filename from changeDir, applies extract, and appends
// "label: <result>" to sections when the file exists and extraction is non-empty.
func appendArtifactSection(sections []string, changeDir, filename, label string, extract func(string) string) []string {
	if data, err := os.ReadFile(filepath.Join(changeDir, filename)); err == nil {
		if s := extract(string(data)); s != "" {
			return append(sections, label+": "+s)
		}
	}
	return sections
}

// buildSummary scans existing artifacts in changeDir and produces a compact
// cumulative context (~500-800 bytes) that carries key decisions forward
// through the pipeline. Non-fatal: returns empty string if no artifacts exist.
func buildSummary(changeDir string, p *Params) string {
	sections := make([]string, 0, 6)

	sections = append(sections, fmt.Sprintf("Change: %s — %s", p.ChangeName, p.Description))
	sections = append(sections, fmt.Sprintf("Stack: %s (%s)", p.Config.Stack.Language, p.Config.Stack.BuildTool))

	// Extract key lines from each artifact if it exists.
	sections = appendArtifactSection(sections, changeDir, "exploration.md", "Exploration", func(s string) string { return extractFirst(s, "##", 3) })
	sections = appendArtifactSection(sections, changeDir, "proposal.md", "Proposal", extractDecisions)
	sections = appendArtifactSection(sections, changeDir, "design.md", "Design", extractDecisions)
	sections = appendArtifactSection(sections, changeDir, "review-report.md", "Review", func(s string) string { return extractFirst(s, "Verdict", 1) })

	return strings.Join(sections, "\n")
}

// extractFirst finds the first line containing keyword after the first heading,
// then returns up to n non-empty content lines following it.
// Used to pull key decisions from artifacts without loading the entire file.
func extractFirst(content, keyword string, maxLines int) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, maxLines)

	collecting := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !collecting && strings.Contains(trimmed, keyword) {
			collecting = true
			// Don't include the header itself — get content after it.
			continue
		}

		if collecting {
			// Skip sub-headers.
			if strings.HasPrefix(trimmed, "#") {
				if len(result) > 0 {
					break // hit next section, stop
				}
				continue
			}
			result = append(result, trimmed)
			if len(result) >= maxLines {
				break
			}
		}
	}

	return strings.Join(result, " ")
}

func isDecisionKey(s string) bool {
	if len(s) == 0 || len(s) > 30 {
		return false
	}
	if strings.ContainsAny(s, " \t") {
		return false
	}
	if strings.HasPrefix(s, "http") || strings.HasPrefix(s, "-") {
		return false
	}
	return true
}

func extractDecisions(content string) string {
	lines := strings.Split(content, "\n")
	kvPairs := make([]string, 0, 5)
	headerLines := make([]string, 0, 3)
	inFence := false
	inDecisionSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			header := strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			inDecisionSection = header == "decisions" || header == "architecture"
			continue
		}

		if inDecisionSection && trimmed != "" {
			headerLines = append(headerLines, trimmed)
			if len(headerLines) >= 3 {
				inDecisionSection = false
			}
			continue
		}

		if !inDecisionSection && strings.Contains(trimmed, ": ") {
			parts := strings.SplitN(trimmed, ": ", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if isDecisionKey(key) && value != "" {
				kvPairs = append(kvPairs, key+": "+value)
				if len(kvPairs) >= 5 {
					break
				}
			}
		}
	}

	if len(kvPairs) > 0 {
		return strings.Join(kvPairs, "; ")
	}
	if len(headerLines) > 0 {
		return strings.Join(headerLines, " ")
	}
	return extractFirst(content, "##", 3)
}

// projectContext returns a compact project overview string with stack info.
func projectContext(p *Params) string {
	return fmt.Sprintf(
		"Project: %s\nLanguage: %s\nBuild Tool: %s\nManifests: %s",
		p.Config.ProjectName,
		p.Config.Stack.Language,
		p.Config.Stack.BuildTool,
		strings.Join(p.Config.Stack.Manifests, ", "),
	)
}

// loadManifestContents reads the actual content of detected manifest files.
// Returns a compact summary with versions and dependencies.
func loadManifestContents(projectDir string, manifests []string) string {
	parts := make([]string, 0, len(manifests))
	for _, m := range manifests {
		data, err := os.ReadFile(filepath.Join(projectDir, m))
		if err != nil {
			continue
		}
		// Cap at 2KB per manifest to keep context lean.
		content := string(data)
		if len(content) > 2048 {
			content = content[:2048] + "\n... (truncated)"
		}
		parts = append(parts, fmt.Sprintf("### %s\n\n```\n%s\n```", m, content))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

// extractCompletedTasks returns a summary of completed task sections.
func extractCompletedTasks(tasks string) string {
	lines := strings.Split(tasks, "\n")
	var completed []string
	var currentSection string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			currentSection = trimmed
			continue
		}
		if strings.HasPrefix(trimmed, "- [x]") {
			task := strings.TrimPrefix(trimmed, "- [x] ")
			if currentSection != "" {
				completed = append(completed, fmt.Sprintf("%s: %s", currentSection, task))
			} else {
				completed = append(completed, task)
			}
		}
	}

	if len(completed) == 0 {
		return "(no tasks completed yet)"
	}
	return strings.Join(completed, "\n")
}
