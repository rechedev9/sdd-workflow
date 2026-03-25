package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func TestWritePending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("# Exploration\n\nFindings here.\n")

	err := WritePending(dir, state.PhaseExplore, content)
	if err != nil {
		t.Fatalf("WritePending: %v", err)
	}

	path := PendingPath(dir, state.PhaseExplore)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pending: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWritePendingSpec(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("# Spec\n\n## Requirements\n- OAuth login\n")

	err := WritePending(dir, state.PhaseSpec, content)
	if err != nil {
		t.Fatalf("WritePending spec: %v", err)
	}

	path := filepath.Join(PendingPath(dir, state.PhaseSpec), PendingFileName(state.PhaseSpec))
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pending spec: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestPendingExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if PendingExists(dir, state.PhaseExplore) {
		t.Error("should not exist before write")
	}

	WritePending(dir, state.PhaseExplore, []byte("test"))

	if !PendingExists(dir, state.PhaseExplore) {
		t.Error("should exist after write")
	}
}

func TestPendingExistsSpecTree(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(PendingPath(dir, state.PhaseSpec), "watch-cli"), 0o755); err != nil {
		t.Fatalf("mkdir pending spec tree: %v", err)
	}
	if PendingExists(dir, state.PhaseSpec) {
		t.Fatal("empty spec pending tree should not count as pending artifact")
	}
	if err := os.WriteFile(filepath.Join(PendingPath(dir, state.PhaseSpec), "watch-cli", "spec.md"), []byte("# Spec\n\n## Requirements\n- Watch\n"), 0o644); err != nil {
		t.Fatalf("write pending spec file: %v", err)
	}
	if !PendingExists(dir, state.PhaseSpec) {
		t.Fatal("spec pending tree should exist after adding a file")
	}
}

func TestPromote(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("# Exploration\n\n## Current State\nLogin page exists.\n\n## Relevant Files\n- login.go\n")

	WritePending(dir, state.PhaseExplore, content)

	promoted, err := Promote(dir, state.PhaseExplore, false)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}

	expected := filepath.Join(dir, "exploration.md")
	if promoted != expected {
		t.Errorf("promoted path = %q, want %q", promoted, expected)
	}

	// Verify promoted file exists with correct content.
	got, err := os.ReadFile(promoted)
	if err != nil {
		t.Fatalf("read promoted: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	// Verify pending file is gone.
	if PendingExists(dir, state.PhaseExplore) {
		t.Error("pending file should be removed after promotion")
	}
}

func TestPromoteSpec(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	contentA := []byte("# Watch CLI\n\n## Requirements\n- OAuth login\n")
	contentB := []byte("# Watch Loop\n\n## Requirements\n- Reassemble\n")
	src := PendingPath(dir, state.PhaseSpec)
	if err := os.MkdirAll(filepath.Join(src, "watch-cli"), 0o755); err != nil {
		t.Fatalf("mkdir watch-cli: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "watch-loop"), 0o755); err != nil {
		t.Fatalf("mkdir watch-loop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "watch-cli", "spec.md"), contentA, 0o644); err != nil {
		t.Fatalf("write watch-cli spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "watch-loop", "spec.md"), contentB, 0o644); err != nil {
		t.Fatalf("write watch-loop spec: %v", err)
	}

	promoted, err := Promote(dir, state.PhaseSpec, false)
	if err != nil {
		t.Fatalf("Promote spec: %v", err)
	}

	if promoted != filepath.Join(dir, "specs") {
		t.Fatalf("promoted path = %q, want %q", promoted, filepath.Join(dir, "specs"))
	}

	got, err := os.ReadFile(filepath.Join(promoted, "watch-cli", "spec.md"))
	if err != nil {
		t.Fatalf("read promoted watch-cli spec: %v", err)
	}
	if string(got) != string(contentA) {
		t.Errorf("content = %q, want %q", got, contentA)
	}
	got, err = os.ReadFile(filepath.Join(promoted, "watch-loop", "spec.md"))
	if err != nil {
		t.Fatalf("read promoted watch-loop spec: %v", err)
	}
	if string(got) != string(contentB) {
		t.Errorf("content = %q, want %q", got, contentB)
	}
	if PendingExists(dir, state.PhaseSpec) {
		t.Error("pending spec tree should be removed after promotion")
	}
}

func TestPromoteSpecReplacesExistingDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dst := filepath.Join(dir, "specs")
	if err := os.MkdirAll(filepath.Join(dst, "obsolete"), 0o755); err != nil {
		t.Fatalf("mkdir obsolete: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "obsolete", "spec.md"), []byte("old"), 0o644); err != nil {
		t.Fatalf("write obsolete spec: %v", err)
	}
	src := PendingPath(dir, state.PhaseSpec)
	if err := os.MkdirAll(filepath.Join(src, "watch-cli"), 0o755); err != nil {
		t.Fatalf("mkdir pending watch-cli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "watch-cli", "spec.md"), []byte("# Spec\n\n## Requirements\n- New\n"), 0o644); err != nil {
		t.Fatalf("write new spec: %v", err)
	}

	promoted, err := Promote(dir, state.PhaseSpec, false)
	if err != nil {
		t.Fatalf("Promote spec replace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(promoted, "obsolete", "spec.md")); !os.IsNotExist(err) {
		t.Fatalf("obsolete spec should be removed, stat err = %v", err)
	}
}

func TestPromoteNoPending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := Promote(dir, state.PhaseExplore, false)
	if err == nil {
		t.Fatal("expected error for missing pending")
	}
	if !errors.Is(err, ErrNoPending) {
		t.Errorf("error = %v, want ErrNoPending", err)
	}
}

func TestRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("# Exploration\n")
	os.WriteFile(filepath.Join(dir, "exploration.md"), content, 0o644)

	got, err := Read(dir, state.PhaseExplore)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestReadMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	_, err := Read(dir, state.PhaseExplore)
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("custom content")
	os.WriteFile(filepath.Join(dir, "custom.md"), content, 0o644)

	got, err := ReadFile(dir, "custom.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create some artifacts.
	os.WriteFile(filepath.Join(dir, "exploration.md"), []byte("explore"), 0o644)
	os.WriteFile(filepath.Join(dir, "proposal.md"), []byte("propose"), 0o644)
	os.WriteFile(filepath.Join(dir, "design.md"), []byte("design content"), 0o644)

	items, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("item count = %d, want 3", len(items))
	}

	// Verify phases are present.
	phases := map[state.Phase]bool{}
	for _, item := range items {
		phases[item.Phase] = true
		if item.Size == 0 {
			t.Errorf("artifact %s has zero size", item.Filename)
		}
	}
	for _, p := range []state.Phase{state.PhaseExplore, state.PhasePropose, state.PhaseDesign} {
		if !phases[p] {
			t.Errorf("missing phase %s in list", p)
		}
	}
}

func TestListWithSpecs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	os.MkdirAll(specsDir, 0o755)
	os.WriteFile(filepath.Join(specsDir, "auth-spec.md"), []byte("auth"), 0o644)
	os.WriteFile(filepath.Join(specsDir, "api-spec.md"), []byte("api"), 0o644)

	items, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("item count = %d, want 2 (spec files)", len(items))
	}
	for _, item := range items {
		if item.Phase != state.PhaseSpec {
			t.Errorf("phase = %s, want spec", item.Phase)
		}
	}
}

func TestList_SpecsDirEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create an empty specs/ directory — List should skip it (len(entries) == 0).
	os.MkdirAll(filepath.Join(dir, "specs"), 0o755)

	items, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("item count = %d, want 0 for empty specs dir", len(items))
	}
}

func TestList_SpecsDirUnreadable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	os.MkdirAll(specsDir, 0o755)
	os.WriteFile(filepath.Join(specsDir, "spec.md"), []byte("spec"), 0o644)

	// Make specs/ unreadable so ReadDir fails — List should skip it.
	os.Chmod(specsDir, 0o000)
	t.Cleanup(func() { os.Chmod(specsDir, 0o755) })

	items, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Spec items should be skipped due to ReadDir error.
	for _, item := range items {
		if item.Phase == state.PhaseSpec {
			t.Error("spec item should not appear when specs/ is unreadable")
		}
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	items, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("item count = %d, want 0", len(items))
	}
}

func TestListPending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	WritePending(dir, state.PhaseExplore, []byte("explore"))
	WritePending(dir, state.PhasePropose, []byte("propose"))
	if err := os.MkdirAll(filepath.Join(PendingPath(dir, state.PhaseSpec), "watch-cli"), 0o755); err != nil {
		t.Fatalf("mkdir spec pending: %v", err)
	}
	if err := os.WriteFile(filepath.Join(PendingPath(dir, state.PhaseSpec), "watch-cli", "spec.md"), []byte("# Spec\n\n## Requirements\n- Watch\n"), 0o644); err != nil {
		t.Fatalf("write spec pending: %v", err)
	}

	items, err := ListPending(dir)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("pending count = %d, want 3", len(items))
	}
	foundSpec := false
	for _, item := range items {
		if item.Filename == filepath.Join("specs", "watch-cli", "spec.md") {
			foundSpec = true
			if item.Phase != state.PhaseSpec {
				t.Fatalf("pending spec phase = %s, want spec", item.Phase)
			}
		}
	}
	if !foundSpec {
		t.Fatal("expected nested spec pending file in listing")
	}
}

func TestListPendingEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	items, err := ListPending(dir)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if items != nil {
		t.Errorf("expected nil for missing .pending dir, got %v", items)
	}
}

func TestArtifactFileNameUnknownPhase(t *testing.T) {
	t.Parallel()
	_, ok := ArtifactFileName(state.Phase("nonexistent"))
	if ok {
		t.Error("expected ok=false for unknown phase")
	}
}

func TestReadUnknownPhase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := Read(dir, state.Phase("nonexistent"))
	if err == nil {
		t.Error("expected error for unknown phase")
	}
}

func TestReadFileMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := ReadFile(dir, "nonexistent.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestPendingFileName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		phase state.Phase
		want  string
	}{
		{state.PhaseExplore, "explore.md"},
		{state.PhasePropose, "propose.md"},
		{state.PhaseSpec, "spec.md"},
		{state.PhaseDesign, "design.md"},
		{state.PhaseTasks, "tasks.md"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.phase), func(t *testing.T) {
			t.Parallel()
			got := PendingFileName(tt.phase)
			if got != tt.want {
				t.Errorf("PendingFileName(%s) = %q, want %q", tt.phase, got, tt.want)
			}
		})
	}
}

func TestPromoteValidationReject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Explore artifact missing required sections.
	WritePending(dir, state.PhaseExplore, []byte("# Exploration\n\nno required sections"))

	_, err := Promote(dir, state.PhaseExplore, false)
	if err == nil {
		t.Fatal("expected validation error for invalid explore artifact")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("error should wrap ErrValidation, got: %v", err)
	}
	// Pending file should still exist (not promoted).
	if !PendingExists(dir, state.PhaseExplore) {
		t.Error("pending file should still exist after rejected promotion")
	}
}

func TestPromoteForceBypass(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Explore artifact missing required sections — but force=true.
	WritePending(dir, state.PhaseExplore, []byte("# Exploration\n\nno required sections"))

	promoted, err := Promote(dir, state.PhaseExplore, true)
	if err != nil {
		t.Fatalf("force promote should succeed, got: %v", err)
	}
	if _, err := os.Stat(promoted); err != nil {
		t.Errorf("promoted file missing: %v", err)
	}
}

func TestPromote_NoArtifactMapping(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Manually create a pending file for an unknown phase.
	pendingDir := filepath.Join(dir, ".pending")
	os.MkdirAll(pendingDir, 0o755)
	os.WriteFile(filepath.Join(pendingDir, "unknown-phase.md"), []byte("content"), 0o644)

	_, err := Promote(dir, state.Phase("unknown-phase"), true)
	if err == nil {
		t.Fatal("expected error for unknown phase with no artifact mapping")
	}
}

func TestPromote_RemoveFails_ReturnsDestWithoutError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := []byte("# Exploration\n\n## Current State\nWorks.\n\n## Relevant Files\n- main.go\n")
	WritePending(dir, state.PhaseExplore, content)

	// Make .pending/ read-only so os.Remove fails.
	pendingDir := filepath.Join(dir, ".pending")
	os.Chmod(pendingDir, 0o555)
	t.Cleanup(func() { os.Chmod(pendingDir, 0o755) })

	dst, err := Promote(dir, state.PhaseExplore, true)
	if err != nil {
		t.Fatalf("expected success even when Remove fails, got: %v", err)
	}
	if dst == "" {
		t.Error("expected non-empty dst path")
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("promoted file missing: %v", err)
	}
}

func TestListPending_ReadDirError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".pending")
	// Create .pending as a file (not a dir) so ReadDir returns an error that isn't NotExist.
	os.WriteFile(pendingDir, []byte("not a dir"), 0o644)

	_, err := ListPending(dir)
	if err == nil {
		t.Fatal("expected error when .pending is a file, not a directory")
	}
}

func TestWritePending_MkdirAllError(t *testing.T) {
	t.Parallel()
	// Create a file where .pending/ should be, so MkdirAll fails.
	root := t.TempDir()
	barrier := filepath.Join(root, ".pending")
	os.WriteFile(barrier, []byte("block"), 0o644)

	err := WritePending(root, state.PhaseExplore, []byte("data"))
	if err == nil {
		t.Fatal("expected error when .pending is a file, not a directory")
	}
}

func TestPromote_WriteFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	WritePending(dir, state.PhaseExplore, []byte("# Exploration\n\n## Current State\nOK.\n\n## Relevant Files\n- a.go\n"))

	// Make the change directory read-only so WriteFile fails.
	os.Chmod(dir, 0o555)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	_, err := Promote(dir, state.PhaseExplore, true)
	if err == nil {
		t.Fatal("expected error when destination directory is read-only")
	}
}

func TestPromote_SpecMkdirAllFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	WritePending(dir, state.PhaseSpec, []byte("spec content"))

	// Make the change directory read-only so creating specs/ fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := Promote(dir, state.PhaseSpec, true)
	if err == nil {
		t.Fatal("expected error when specs/ cannot be created")
	}
}

func TestPromoteAllPhases(t *testing.T) {
	t.Parallel()
	// Verify every phase with an artifact mapping can be promoted.
	phases := []state.Phase{
		state.PhaseExplore, state.PhasePropose, state.PhaseDesign,
		state.PhaseTasks, state.PhaseReview, state.PhaseVerify,
		state.PhaseClean, state.PhaseArchive,
	}
	for _, phase := range phases {
		phase := phase
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			WritePending(dir, phase, []byte("content for "+string(phase)))

			promoted, err := Promote(dir, phase, true)
			if err != nil {
				t.Fatalf("Promote(%s): %v", phase, err)
			}
			if _, err := os.Stat(promoted); err != nil {
				t.Errorf("promoted file missing: %v", err)
			}
		})
	}
}

func TestWritePending_WriteError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Make .pending/ read-only so WriteFile fails.
	if err := os.Chmod(pendingDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(pendingDir, 0o755) })

	err := WritePending(dir, state.PhaseExplore, []byte("content"))
	if err == nil {
		t.Fatal("expected error writing to read-only .pending dir")
	}
}

func TestPromote_ReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create pending file then make it unreadable.
	if err := WritePending(dir, state.PhaseExplore, []byte("content")); err != nil {
		t.Fatalf("setup WritePending: %v", err)
	}
	pendingFile := PendingPath(dir, state.PhaseExplore)
	if err := os.Chmod(pendingFile, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(pendingFile, 0o644) })

	_, err := Promote(dir, state.PhaseExplore, true)
	if err == nil {
		t.Fatal("expected error reading unreadable pending file")
	}
}
