package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzConfigLoad(f *testing.F) {
	f.Add([]byte(`project_name: test
stack:
  language: go
  build_tool: go
`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not yaml`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Skip()
		}
		Load(path) // must not panic
	})
}
