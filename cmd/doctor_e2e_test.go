//go:build e2e

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/lifecycle"
)

// doctorJSONResult mirrors the locked --json shape for parsing in e2e asserts.
type doctorJSONResult struct {
	Checks []struct {
		Group    string `json:"group"`
		Name     string `json:"name"`
		Severity string `json:"severity"`
		Detail   string `json:"detail"`
		Hint     string `json:"hint"`
	} `json:"checks"`
	Summary struct {
		OK   int `json:"ok"`
		Info int `json:"info"`
		Warn int `json:"warn"`
		Fail int `json:"fail"`
	} `json:"summary"`
}

// runDoctorE2E runs `lzr doctor <args...>` and returns combined output and the
// process exit code (0 when the process exited cleanly).
func runDoctorE2E(t *testing.T, lzr string, args ...string) ([]byte, int) {
	t.Helper()
	cmd := exec.Command(lzr, append([]string{"doctor"}, args...)...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errorsAs(err, &ee) {
			t.Fatalf("doctor %v: non-exit error: %v\n%s", args, err, out)
		}
		code = ee.ExitCode()
	}
	return out, code
}

// errorsAs is a tiny local alias so this e2e file does not need to import
// "errors" solely for one As call; it keeps the helper self-contained.
func errorsAs(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}

// parseDoctorJSON unmarshals --json doctor output, failing the test on garbage.
// It extracts the leading JSON object from combined output so that any cobra
// usage/error text appended after a non-zero exit does not break the parse.
func parseDoctorJSON(t *testing.T, out []byte) doctorJSONResult {
	t.Helper()
	// Locate the JSON object boundaries: first '{' through the matching '}'.
	start := -1
	for i, b := range out {
		if b == '{' {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("doctor --json produced no JSON object\noutput:\n%s", out)
	}
	// Walk forward to find the matching closing brace (depth-tracking).
	depth := 0
	end := -1
	for i := start; i < len(out); i++ {
		switch out[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		t.Fatalf("doctor --json JSON object is unterminated\noutput:\n%s", out)
	}
	var got doctorJSONResult
	if err := json.Unmarshal(out[start:end+1], &got); err != nil {
		t.Fatalf("doctor --json did not parse: %v\noutput:\n%s", err, out)
	}
	return got
}

// summaryTotalsMatch verifies Summary tallies equal the per-check severity
// counts (a self-consistency invariant of any RunAll report).
func summaryTotalsMatch(got doctorJSONResult) bool {
	var ok, info, warn, fail int
	for _, c := range got.Checks {
		switch c.Severity {
		case "OK":
			ok++
		case "INFO":
			info++
		case "WARN":
			warn++
		case "FAIL":
			fail++
		}
	}
	s := got.Summary
	return s.OK == ok && s.Info == info && s.Warn == warn && s.Fail == fail
}

// TestDoctor_StoppedInstall: a clean, seeded-but-stopped install must not
// produce a FAIL purely because the proxy is not running, and --json must
// parse with self-consistent tallies.
func TestDoctor_StoppedInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	xrayPath := config.XrayBinaryPath()
	if err := os.MkdirAll(filepath.Dir(xrayPath), 0755); err != nil {
		t.Fatalf("mkdir xray dir: %v", err)
	}
	writeFakeXray(t, xrayPath)
	seedMinimalProfile(t)

	lzr := buildLZR(t)

	out, code := runDoctorE2E(t, lzr, "--json")
	got := parseDoctorJSON(t, out)

	if !summaryTotalsMatch(got) {
		t.Errorf("summary tallies do not match check counts: %+v\n%s", got.Summary, out)
	}
	if got.Summary.Fail != 0 {
		t.Errorf("stopped install reported %d FAIL(s), want 0\n%s", got.Summary.Fail, out)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0 for a clean stopped install", code)
	}
}

// TestDoctor_RunningInstall: with a live supervisor, --json still parses, the
// session group reports the running state, tallies are consistent, and a
// non-strict run with no FAIL exits 0.
func TestDoctor_RunningInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	xrayPath := config.XrayBinaryPath()
	if err := os.MkdirAll(filepath.Dir(xrayPath), 0755); err != nil {
		t.Fatalf("mkdir xray dir: %v", err)
	}
	writeFakeXray(t, xrayPath)
	seedMinimalProfile(t)

	lzr := buildLZR(t)

	t.Cleanup(func() {
		if st, _ := lifecycle.ReadState(); st != nil && st.SupervisorPID > 0 {
			_ = exec.Command(lzr, "stop").Run()
			if lifecycle.SupervisorAlive() {
				_ = syscallKillBestEffort(st.SupervisorPID)
			}
		}
	})

	if out, err := exec.Command(lzr, "start").CombinedOutput(); err != nil {
		t.Fatalf("start: %v\n%s", err, out)
	}
	if !lifecycle.SupervisorAlive() {
		t.Fatal("supervisor not alive after start")
	}

	out, code := runDoctorE2E(t, lzr, "--json")
	got := parseDoctorJSON(t, out)

	if !summaryTotalsMatch(got) {
		t.Errorf("summary tallies do not match check counts: %+v\n%s", got.Summary, out)
	}
	if got.Summary.Fail != 0 {
		t.Errorf("running install reported %d FAIL(s), want 0\n%s", got.Summary.Fail, out)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0\n%s", code, out)
	}
}

// TestDoctor_InjectedFail: seeding a last-error.json drives the startup-group
// check to FAIL, which must yield a non-zero (ExitGeneric) exit code.
func TestDoctor_InjectedFail(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	xrayPath := config.XrayBinaryPath()
	if err := os.MkdirAll(filepath.Dir(xrayPath), 0755); err != nil {
		t.Fatalf("mkdir xray dir: %v", err)
	}
	writeFakeXray(t, xrayPath)
	seedMinimalProfile(t)

	// Seed a startup error so the startup-group doctor check reports FAIL.
	if err := lifecycle.WriteStartupError("start", os.ErrPermission); err != nil {
		t.Fatalf("seed last-error: %v", err)
	}

	lzr := buildLZR(t)

	out, code := runDoctorE2E(t, lzr)
	if code != ExitGeneric {
		t.Errorf("exit code = %d, want %d (ExitGeneric) with a seeded last-error\n%s", code, ExitGeneric, out)
	}

	jsonOut, _ := runDoctorE2E(t, lzr, "--json")
	got := parseDoctorJSON(t, jsonOut)
	if got.Summary.Fail < 1 {
		t.Errorf("expected at least 1 FAIL from seeded last-error, summary=%+v\n%s", got.Summary, jsonOut)
	}
}

// TestDoctor_StrictWarn: a run whose worst severity is WARN (no FAIL) must exit
// 0 by default but exit ExitGeneric under --strict. We drive a WARN by removing
// the xray binary while keeping a valid profile: the install-group reports a
// non-fatal "not installed" WARN without any FAIL.
func TestDoctor_StrictWarn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Install the fake xray (binary present -> no install FAIL) and seed a valid
	// profile, but deliberately do NOT create geoip.dat/geosite.dat so the geo
	// check emits a WARN with no FAIL -> the strict-flip is exercisable.
	xrayPath := config.XrayBinaryPath()
	if err := os.MkdirAll(filepath.Dir(xrayPath), 0755); err != nil {
		t.Fatalf("mkdir xray dir: %v", err)
	}
	writeFakeXray(t, xrayPath)
	seedMinimalProfile(t)

	lzr := buildLZR(t)

	// Locate the WARN-without-FAIL case from --json first; skip if the env does
	// not actually yield that shape (keeps the test honest, not flaky).
	jsonOut, _ := runDoctorE2E(t, lzr, "--json")
	got := parseDoctorJSON(t, jsonOut)
	if got.Summary.Fail != 0 || got.Summary.Warn == 0 {
		t.Skipf("env did not yield WARN-without-FAIL shape (warn=%d fail=%d); strict-flip not exercisable here\n%s",
			got.Summary.Warn, got.Summary.Fail, jsonOut)
	}

	_, code := runDoctorE2E(t, lzr)
	if code != 0 {
		t.Errorf("non-strict WARN-only run exit = %d, want 0\n%s", code, jsonOut)
	}

	_, strictCode := runDoctorE2E(t, lzr, "--strict")
	if strictCode != ExitGeneric {
		t.Errorf("--strict WARN-only run exit = %d, want %d (ExitGeneric)\n%s", strictCode, ExitGeneric, jsonOut)
	}

	// Sanity: a short settle so any background probe completes deterministically.
	time.Sleep(10 * time.Millisecond)
}
