package state

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzStateLoad(f *testing.F) {
	f.Add([]byte(`{"name":"test","description":"d","current_phase":"explore","phases":{"explore":"pending","propose":"pending","spec":"pending","design":"pending","tasks":"pending","apply":"pending","review":"pending","verify":"pending","clean":"pending","ship":"pending","archive":"pending"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "state.json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Skip()
		}
		Load(path) // must not panic
	})
}
