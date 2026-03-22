package artifacts

import (
	"testing"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

func FuzzValidate(f *testing.F) {
	phases := state.AllPhases()

	// Valid seeds per phase.
	f.Add(uint8(0), []byte("## Current State\n\n## Relevant Files\n")) // explore
	f.Add(uint8(1), []byte("## Intent\n\n## Scope\n"))                 // propose
	f.Add(uint8(2), []byte("## Requirements\n"))                       // spec
	f.Add(uint8(3), []byte("## Architecture\n"))                       // design
	f.Add(uint8(4), []byte("- [ ] Task one\n- [x] Task two\n"))        // tasks
	f.Add(uint8(5), []byte("- [ ] Apply task\n"))                      // apply
	f.Add(uint8(6), []byte("## Review\nmain.go:42\nPASS\n"))           // review
	f.Add(uint8(7), []byte("## Verify\n"))                             // verify (no rules)
	f.Add(uint8(8), []byte("## Clean\n"))                              // clean
	f.Add(uint8(9), []byte("## Archive\n"))                            // archive (no rules)

	// Edge cases.
	f.Add(uint8(0), []byte{})                                            // empty
	f.Add(uint8(6), []byte("## \n.go:\nPAS\n"))                          // partial matches
	f.Add(uint8(6), []byte("## Review\na.b:0\nFAILURE contains FAIL\n")) // substring overlap
	f.Add(uint8(4), []byte("-[\n"))                                      // malformed checkbox
	f.Add(uint8(2), []byte("##NoSpace\n"))                               // heading without space
	f.Add(uint8(0), []byte("# Current State\n"))                         // single hash (not ##)
	f.Add(uint8(6), []byte("\x00\xff\xfe"))                              // binary content
	f.Add(uint8(2), []byte("##  \n"))                                    // heading with trailing space
	f.Add(uint8(6), []byte("## X\nAPPROVED\nfile.go:1\n"))               // all review rules met
	f.Add(uint8(6), []byte("## X\nREJECTED\nserver.go:999\n"))           // alternate verdict

	f.Fuzz(func(t *testing.T, phaseIdx uint8, data []byte) {
		ph := phases[int(phaseIdx)%len(phases)]
		Validate(ph, data) // must not panic
	})
}
