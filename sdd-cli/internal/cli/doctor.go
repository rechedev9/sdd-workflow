package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rechedev9/shenronSDD/sdd-cli/internal/artifacts"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/config"
	sddctx "github.com/rechedev9/shenronSDD/sdd-cli/internal/context"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/errlog"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/phase"
	"github.com/rechedev9/shenronSDD/sdd-cli/internal/state"
)

// CheckResult holds the outcome of a single diagnostic check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func checkConfig(configPath string) (CheckResult, *config.Config) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return checkFail("config", err.Error()), nil
	}
	if cfg.Version != 0 && cfg.Version != config.ConfigVersion {
		msg := fmt.Sprintf("config version %d, expected %d", cfg.Version, config.ConfigVersion)
		return checkWarn("config", msg), cfg
	}
	return checkPass("config", fmt.Sprintf("config.yaml v%d loaded", cfg.Version)), cfg
}

func checkCache(changesDir string, cfg *config.Config) CheckResult {
	if _, err := os.ReadDir(changesDir); err != nil {
		return checkWarn("cache", "cannot read changes directory")
	}
	skillsPath := ""
	if cfg != nil {
		skillsPath = cfg.SkillsPath
	}
	total := 0
	eachChangeDir(changesDir, func(changeDir string) {
		n, _ := sddctx.CheckCacheIntegrity(changeDir, skillsPath)
		total += n
	})
	if total > 0 {
		return checkWarn("cache", fmt.Sprintf("%d stale cache entry(s)", total))
	}
	return checkPass("cache", "all cache entries current")
}

func checkOrphanedPending(changesDir string) CheckResult {
	count := 0
	eachChangeDir(changesDir, func(changeDir string) {
		pendingDir := filepath.Join(changeDir, ".pending")
		pfiles, err := os.ReadDir(pendingDir)
		if err != nil {
			return
		}
		for _, pf := range pfiles {
			if pf.IsDir() || !strings.HasSuffix(pf.Name(), ".md") {
				continue
			}
			ph := state.Phase(strings.TrimSuffix(pf.Name(), ".md"))
			artifactFile, ok := artifacts.ArtifactFileName(ph)
			if !ok {
				continue
			}
			// Skip phases that reuse a predecessor's artifact (e.g. apply → tasks.md).
			// That artifact exists from the predecessor phase and is not evidence of
			// this phase being promoted.
			if desc, ok := phase.DefaultRegistry.Get(string(ph)); ok && desc.RecoverSkip {
				continue
			}
			// spec promotes into specs/{pendingFileName}; others promote directly.
			var promoted string
			if artifactFile == "specs" {
				promoted = filepath.Join(changeDir, "specs", pf.Name())
			} else {
				promoted = filepath.Join(changeDir, artifactFile)
			}
			if _, err := os.Stat(promoted); err == nil {
				count++
			}
		}
	})
	if count > 0 {
		return checkWarn("orphaned_pending", fmt.Sprintf("%d orphaned .pending file(s)", count))
	}
	return checkPass("orphaned_pending", "")
}

func checkSkillsPath(cfg *config.Config) CheckResult {
	if cfg == nil {
		return checkWarn("skills_path", "skipped: config unavailable")
	}
	if cfg.SkillsPath == "" {
		return checkWarn("skills_path", "no skills_path configured — using embedded prompts")
	}
	if _, err := os.Stat(cfg.SkillsPath); err != nil {
		return checkFail("skills_path", fmt.Sprintf("skills directory not found: %s", cfg.SkillsPath))
	}
	phases := state.AllPhases()
	present := 0
	for _, p := range phases {
		skillPath := filepath.Join(cfg.SkillsPath, "sdd-"+string(p), "SKILL.md")
		if _, err := os.Stat(skillPath); err == nil {
			present++
		}
	}
	msg := fmt.Sprintf("%d/%d SKILL.md files present", present, len(phases))
	if present < len(phases) {
		return checkWarn("skills_path", msg)
	}
	return checkPass("skills_path", msg)
}

func checkBuildTools(cfg *config.Config) CheckResult {
	if cfg == nil {
		return checkWarn("build_tools", "skipped: config unavailable")
	}
	cmds := []string{cfg.Commands.Build, cfg.Commands.Test, cfg.Commands.Lint, cfg.Commands.Format}
	missing := make([]string, 0, len(cmds))
	seen := map[string]bool{}
	for _, cmd := range cmds {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		bin := cmd
		if i := strings.IndexAny(cmd, " \t"); i > 0 {
			bin = cmd[:i]
		}
		if seen[bin] {
			continue
		}
		seen[bin] = true
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		return checkFail("build_tools", fmt.Sprintf("not in PATH: %s", strings.Join(missing, ", ")))
	}
	return checkPass("build_tools", "all build commands found")
}

func checkErrors(cwd string) CheckResult {
	log := errlog.Load(cwd)
	if len(log.Entries) == 0 {
		return checkPass("errors", "no recorded errors")
	}
	recurring := log.RecurringFingerprints(3)
	if len(recurring) > 0 {
		return checkWarn("errors", fmt.Sprintf("%d recurring error pattern(s); run 'sdd errors' for details", len(recurring)))
	}
	return checkPass("errors", fmt.Sprintf("%d error(s) recorded, no recurring patterns", len(log.Entries)))
}

func checkPprof() CheckResult {
	val := os.Getenv("SDD_PPROF")
	if val == "" {
		return checkPass("pprof", "SDD_PPROF not set (no profiling)")
	}
	return checkPass("pprof", fmt.Sprintf("SDD_PPROF=%s", val))
}

func checkPass(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: "pass", Message: msg}
}

func checkWarn(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: "warn", Message: msg}
}

func checkFail(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: "fail", Message: msg}
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) error {
	jsonOut := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOut = true
		default:
			return errUnknownFlag(arg)
		}
	}

	cwd, err := getCWD(stderr, "doctor")
	if err != nil {
		return err
	}

	configPath := openspecConfig(cwd)
	changesDir := openspecChanges(cwd)

	configResult, cfg := checkConfig(configPath)
	checks := []CheckResult{
		configResult,
		checkCache(changesDir, cfg),
		checkOrphanedPending(changesDir),
		checkSkillsPath(cfg),
		checkBuildTools(cfg),
		checkErrors(cwd),
		checkPprof(),
	}

	status := aggregateStatus(checks)

	if jsonOut {
		out := struct {
			Command string        `json:"command"`
			Status  string        `json:"status"`
			Checks  []CheckResult `json:"checks"`
		}{
			Command: "doctor",
			Status:  status,
			Checks:  checks,
		}
		writeJSON(stdout, out)
	} else {
		printDoctorTable(stdout, checks)
	}

	if status != "fail" {
		return nil
	}
	failCount := 0
	for _, c := range checks {
		if c.Status == "fail" {
			failCount++
		}
	}
	return fmt.Errorf("doctor: %d check(s) failed", failCount)
}

func aggregateStatus(checks []CheckResult) string {
	worst := "pass"
	for _, c := range checks {
		switch c.Status {
		case "fail":
			return "fail"
		case "warn":
			worst = "warn"
		}
	}
	return worst
}

func printDoctorTable(w io.Writer, checks []CheckResult) {
	maxName := 0
	for _, c := range checks {
		if len(c.Name) > maxName {
			maxName = len(c.Name)
		}
	}
	fmt.Fprintln(w, "sdd doctor")
	for _, c := range checks {
		if c.Message != "" {
			fmt.Fprintf(w, "  %-*s  %-4s  %s\n", maxName, c.Name, c.Status, c.Message)
		} else {
			fmt.Fprintf(w, "  %-*s  %s\n", maxName, c.Name, c.Status)
		}
	}
}
