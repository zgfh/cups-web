package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestCanonMaintenancePayload(t *testing.T) {
	now := time.Date(2026, time.November, 4, 13, 0, 0, 0, time.UTC)
	for action, want := range map[string]string{
		"nozzle-check":       "@TestPrint=NozzleCheck",
		"clean":              "@Cleaning=1ALL",
		"deep-clean":         "@Cleaning=2ALL",
		"roller-clean":       "@RollerCleaning=Roller",
		"bottom-plate-clean": "@RollerCleaning=Platen",
	} {
		payload := canonMaintenancePayload(action, now)
		if !bytes.Contains(payload, []byte(want)) {
			t.Fatalf("%s payload does not contain %q", action, want)
		}
		if !bytes.Contains(payload, []byte("SetTime=20261104130000")) {
			t.Fatalf("%s payload has no expected timestamp", action)
		}
		if !bytes.Contains(payload, []byte("<ivec:operation>StartJob</ivec:operation>")) || !bytes.Contains(payload, []byte("<ivec:operation>EndJob</ivec:operation>")) {
			t.Fatalf("%s payload has incomplete Canon job envelope", action)
		}
	}
}

func TestGenerateMaintenanceTestPage(t *testing.T) {
	path, cleanup, err := generateMaintenanceTestPage()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1024 {
		t.Fatalf("unexpected test page output: %d bytes", len(data))
	}
}
