package state

import (
	"fmt"
	"strconv"
	"strings"
)

func ResolvePhase(input string) (Phase, error) {
	phases := AllPhases()

	if idx, err := strconv.Atoi(input); err == nil {
		if idx < 0 || idx >= len(phases) {
			return "", fmt.Errorf("phase index out of range: %s (valid: 0-%d)", input, len(phases)-1)
		}
		return phases[idx], nil
	}

	// Single pass: check exact match first, then collect prefix matches.
	lower := strings.ToLower(input)
	var matches []string
	for _, p := range phases {
		// Phase names are already lowercase; no need to ToLower(p).
		if string(p) == lower {
			return p, nil // exact match wins immediately
		}
		if strings.HasPrefix(string(p), lower) {
			matches = append(matches, string(p))
		}
	}
	switch len(matches) {
	case 1:
		return Phase(matches[0]), nil
	case 0:
		return "", fmt.Errorf("unknown phase: %q", input)
	default:
		return "", fmt.Errorf("ambiguous phase prefix %q: matches %s", input, strings.Join(matches, ", "))
	}
}
