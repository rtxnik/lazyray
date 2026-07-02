// cmd/doctor_test.go
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/doctor"
)

// withDoctorReport swaps the package-level doctorRunAll seam for a fake.
func withDoctorReport(t *testing.T, rep doctor.Report) {
	t.Helper()
	prev := doctorRunAll
	doctorRunAll = func(_ context.Context, _ *doctor.Env) doctor.Report { return rep }
	t.Cleanup(func() { doctorRunAll = prev })
}

func reportWith(results ...doctor.Result) doctor.Report {
	rep := doctor.Report{Checks: results}
	for _, r := range results {
		switch r.Severity {
		case doctor.SeverityOK:
			rep.Summary.OK++
		case doctor.SeverityInfo:
			rep.Summary.Info++
		case doctor.SeverityWarn:
			rep.Summary.Warn++
		case doctor.SeverityFail:
			rep.Summary.Fail++
		}
	}
	return rep
}

func runDoctor(t *testing.T, jsonFlag, strictFlag bool) (*bytes.Buffer, error) {
	t.Helper()
	prevJSON, prevStrict := doctorJSON, doctorStrict
	doctorJSON, doctorStrict = jsonFlag, strictFlag
	t.Cleanup(func() { doctorJSON, doctorStrict = prevJSON, prevStrict })

	var buf bytes.Buffer
	doctorCmd.SetOut(&buf)
	doctorCmd.SetErr(&buf)
	err := doctorCmd.RunE(doctorCmd, nil)
	return &buf, err
}

func TestDoctor_ExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		report   doctor.Report
		strict   bool
		wantCode int // -1 means "expect nil error"
	}{
		{
			name:     "all ok -> nil",
			report:   reportWith(doctor.Result{Group: "install", Name: "binary", Severity: doctor.SeverityOK, Detail: "found"}),
			wantCode: -1,
		},
		{
			name: "warn without strict -> nil",
			report: reportWith(
				doctor.Result{Group: "routing", Name: "proxy", Severity: doctor.SeverityWarn, Detail: "headless"},
			),
			wantCode: -1,
		},
		{
			name: "warn with strict -> ExitGeneric",
			report: reportWith(
				doctor.Result{Group: "routing", Name: "proxy", Severity: doctor.SeverityWarn, Detail: "headless"},
			),
			strict:   true,
			wantCode: ExitGeneric,
		},
		{
			name: "fail -> ExitGeneric regardless of strict",
			report: reportWith(
				doctor.Result{Group: "startup", Name: "last-error", Severity: doctor.SeverityFail, Detail: "boom"},
			),
			wantCode: ExitGeneric,
		},
		{
			name: "fail under strict -> still ExitGeneric",
			report: reportWith(
				doctor.Result{Group: "startup", Name: "last-error", Severity: doctor.SeverityFail, Detail: "boom"},
				doctor.Result{Group: "routing", Name: "proxy", Severity: doctor.SeverityWarn, Detail: "headless"},
			),
			strict:   true,
			wantCode: ExitGeneric,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDoctorReport(t, tt.report)
			_, err := runDoctor(t, false, tt.strict)

			if tt.wantCode == -1 {
				if err != nil {
					t.Fatalf("want nil error, got %v", err)
				}
				return
			}
			var ee *ExitError
			if !errors.As(err, &ee) {
				t.Fatalf("want *ExitError, got %T (%v)", err, err)
			}
			if ee.Code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", ee.Code, tt.wantCode)
			}
		})
	}
}

func TestDoctor_JSONOutput(t *testing.T) {
	rep := reportWith(
		doctor.Result{Group: "install", Name: "binary", Severity: doctor.SeverityOK, Detail: "found", Hint: ""},
		doctor.Result{Group: "routing", Name: "proxy", Severity: doctor.SeverityWarn, Detail: "headless", Hint: "set http_proxy"},
		doctor.Result{Group: "config", Name: "profile", Severity: doctor.SeverityInfo, Detail: "1 profile"},
	)
	withDoctorReport(t, rep)

	buf, err := runDoctor(t, true, false)
	if err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	var got struct {
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
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON did not parse: %v\noutput:\n%s", err, buf.String())
	}

	if len(got.Checks) != 3 {
		t.Fatalf("checks len = %d, want 3", len(got.Checks))
	}
	if got.Checks[1].Severity != "WARN" {
		t.Errorf("checks[1].severity = %q, want WARN", got.Checks[1].Severity)
	}
	if got.Checks[1].Hint != "set http_proxy" {
		t.Errorf("checks[1].hint = %q, want %q", got.Checks[1].Hint, "set http_proxy")
	}
	if got.Summary.OK != 1 || got.Summary.Info != 1 || got.Summary.Warn != 1 || got.Summary.Fail != 0 {
		t.Errorf("summary = %+v, want ok=1 info=1 warn=1 fail=0", got.Summary)
	}
}

func TestDoctor_HumanGroupsAndSummary(t *testing.T) {
	rep := reportWith(
		doctor.Result{Group: "install", Name: "binary", Severity: doctor.SeverityOK, Detail: "found"},
		doctor.Result{Group: "routing", Name: "proxy", Severity: doctor.SeverityWarn, Detail: "headless", Hint: "set http_proxy"},
	)
	withDoctorReport(t, rep)

	buf, err := runDoctor(t, false, false)
	if err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"install", "routing", "[ OK ]", "[WARN]", "set http_proxy", "1 OK", "1 WARN"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("human output missing %q\nfull output:\n%s", want, out)
		}
	}
}
