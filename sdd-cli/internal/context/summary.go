package context

import (
	"fmt"
	"io"
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

	sections = append(sections, "Change: "+p.ChangeName+" — "+p.Description)
	sections = append(sections, "Stack: "+p.Config.Stack.Language+" ("+p.Config.Stack.BuildTool+")")

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
	result := make([]string, 0, maxLines)

	collecting := false
	for line := range strings.Lines(content) {
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
	kvPairs := make([]string, 0, 5)
	headerLines := make([]string, 0, 3)
	inFence := false
	inDecisionSection := false

	for line := range strings.Lines(content) {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		if strings.HasPrefix(trimmed, "## ") {
			header := strings.TrimPrefix(trimmed, "## ")
			inDecisionSection = strings.EqualFold(header, "decisions") || strings.EqualFold(header, "architecture")
			continue
		}

		if inDecisionSection && trimmed != "" {
			headerLines = append(headerLines, trimmed)
			if len(headerLines) >= 3 {
				inDecisionSection = false
			}
			continue
		}

		if !inDecisionSection {
			if key, value, found := strings.Cut(trimmed, ": "); found {
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				if isDecisionKey(key) && value != "" {
					kvPairs = append(kvPairs, key+": "+value)
					if len(kvPairs) >= 5 {
						break
					}
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

// compactSpecs extracts only headings and MUST/SHOULD/GIVEN/WHEN/THEN lines from specs.
// Reduces a full spec to ~20% of its size while keeping acceptance criteria.
func compactSpecs(specs string) string {
	var b strings.Builder
	for line := range strings.Lines(specs) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "MUST") ||
			strings.HasPrefix(trimmed, "SHOULD") ||
			strings.HasPrefix(trimmed, "- MUST") ||
			strings.HasPrefix(trimmed, "- SHOULD") ||
			strings.Contains(trimmed, "GIVEN") ||
			strings.Contains(trimmed, "WHEN") ||
			strings.Contains(trimmed, "THEN") {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// compactDesign extracts only the Decisions/Architecture section from design.md.
func compactDesign(content string) string {
	return extractDecisions(content)
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

// manifestReadLimit is the max bytes read per manifest file.
// Reading one extra byte lets us detect truncation without reading the whole file.
const manifestReadLimit = 2048

// loadManifestContents reads the actual content of detected manifest files.
// Returns a compact summary with versions and dependencies.
// Reads at most manifestReadLimit bytes per file to avoid loading large lock files.
func loadManifestContents(projectDir string, manifests []string) string {
	parts := make([]string, 0, len(manifests))
	buf := make([]byte, manifestReadLimit+1) // +1 to detect truncation
	for _, m := range manifests {
		f, err := os.Open(filepath.Join(projectDir, m))
		if err != nil {
			continue
		}
		n, _ := io.ReadFull(f, buf)
		f.Close()
		if n == 0 {
			continue
		}
		var content string
		if n > manifestReadLimit {
			content = string(buf[:manifestReadLimit]) + "\n... (truncated)"
		} else {
			content = string(buf[:n])
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
	completed := make([]string, 0, 16)
	var currentSection string

	for line := range strings.Lines(tasks) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "##") {
			currentSection = trimmed
			continue
		}
		if strings.HasPrefix(trimmed, "- [x]") {
			task := strings.TrimLeft(strings.TrimPrefix(trimmed, "- [x]"), " ")
			if currentSection != "" {
				completed = append(completed, currentSection+": "+task)
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
