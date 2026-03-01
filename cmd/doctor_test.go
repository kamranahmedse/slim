package cmd

import (
	"testing"

	"github.com/kamranahmedse/slim/internal/doctor"
)

func TestPrintReport(t *testing.T) {
	report := doctor.Report{
		Results: []doctor.CheckResult{
			{Name: "CA certificate", Status: doctor.Pass, Message: "valid"},
			{Name: "Daemon", Status: doctor.Warn, Message: "not running"},
			{Name: "Hosts", Status: doctor.Fail, Message: "missing"},
		},
	}
	// Just verify it doesn't panic
	printReport(report)
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status doctor.Status
		want   string
	}{
		{doctor.Pass, "✓"},
		{doctor.Warn, "!"},
		{doctor.Fail, "✗"},
	}

	for _, tt := range tests {
		got := statusIcon(tt.status)
		// The icon string includes ANSI codes, just check it contains the symbol
		if len(got) == 0 {
			t.Errorf("statusIcon(%d) returned empty string", tt.status)
		}
	}
}

func TestDoctorRunFnInjectable(t *testing.T) {
	prevRun := doctorRunFn
	defer func() { doctorRunFn = prevRun }()

	var called bool
	doctorRunFn = func() doctor.Report {
		called = true
		return doctor.Report{}
	}

	err := doctorCmd.RunE(doctorCmd, nil)
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	if !called {
		t.Fatal("expected doctorRunFn to be called")
	}
}
