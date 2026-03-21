package context

import "testing"

func FuzzExtractDecisions(f *testing.F) {
	f.Add("approach: middleware\nfallback: noop")
	f.Add("## Decisions\nUse adapter pattern\nNo ORM")
	f.Add("```\nkey: val\n```\nother: x")
	f.Add("# Title\n## Section\nFirst line")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		extractDecisions(input) // must not panic
	})
}

func FuzzExtractFirst(f *testing.F) {
	f.Add("# Title\n## Section\nContent line one\nContent line two", "##", 3)
	f.Add("## Heading\nSome text", "Heading", 1)
	f.Add("", "##", 3)

	f.Fuzz(func(t *testing.T, content string, keyword string, maxLines int) {
		if maxLines < 0 || maxLines > 100 {
			t.Skip()
		}
		extractFirst(content, keyword, maxLines) // must not panic
	})
}

func FuzzExtractCompletedTasks(f *testing.F) {
	f.Add("## Phase 1\n- [x] Done task\n- [ ] Pending")
	f.Add("- [ ] Nothing done")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		extractCompletedTasks(input) // must not panic
	})
}

func FuzzExtractCurrentTask(f *testing.F) {
	f.Add("## Phase 1\n- [x] Done\n- [ ] Next task\n## Phase 2\n- [ ] Later")
	f.Add("- [ ] Only task")
	f.Add("## All done\n- [x] Task 1\n- [x] Task 2")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		extractCurrentTask(input) // must not panic
	})
}
