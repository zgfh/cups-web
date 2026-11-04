package main

import (
	"bytes"
	"testing"
	"time"
)

func TestCanonMaintenancePayload(t *testing.T) {
	now := time.Date(2026, time.November, 4, 13, 0, 0, 0, time.UTC)
	for action, want := range map[string]string{
		"nozzle-check": "@TestPrint=NozzleCheck",
		"clean":        "@Cleaning=1ALL",
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
