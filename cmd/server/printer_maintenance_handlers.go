package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	canonMaintenanceDefaultPrinter = "Canon_MG2500"
	canonMaintenanceTimeout        = 30 * time.Second
)

type printerMaintenanceRequest struct {
	Action string `json:"action"`
}

// printerMaintenanceHandler sends a small, fixed Canon BJL maintenance job
// through the local CUPS queue.  Do not turn this into a generic raw-print
// endpoint: raw data can change device state and bypass all driver safeguards.
// Commands are restricted to the Canon MG2500 maintenance vocabulary.  Each
// one is a fixed byte sequence selected below, never client-supplied raw data.
func printerMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	var req printerMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	action := strings.TrimSpace(req.Action)
	var title string
	switch action {
	case "nozzle-check":
		title = "Canon nozzle check"
	case "clean":
		title = "Canon head cleaning"
	case "deep-clean":
		title = "Canon deep head cleaning"
	case "roller-clean":
		title = "Canon paper feed roller cleaning"
	case "bottom-plate-clean":
		title = "Canon bottom plate cleaning"
	default:
		writeJSONError(w, http.StatusBadRequest, "unsupported maintenance action")
		return
	}

	printer := strings.TrimSpace(os.Getenv("CANON_MAINTENANCE_PRINTER"))
	if printer == "" {
		printer = canonMaintenanceDefaultPrinter
	}
	payload := canonMaintenancePayload(action, time.Now())

	ctx, cancel := context.WithTimeout(r.Context(), canonMaintenanceTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "lp", "-d", printer, "-o", "raw", "-t", title, "-")
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		writeJSONError(w, http.StatusGatewayTimeout, "maintenance job submission timed out")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to submit maintenance job")
		return
	}

	job := strings.TrimSpace(string(output))
	writeJSON(w, map[string]string{
		"ok":      "true",
		"action":  action,
		"printer": printer,
		"job":     job,
	})
}

func canonMaintenancePayload(action string, now time.Time) []byte {
	commands := map[string]string{
		"nozzle-check":       "@TestPrint=NozzleCheck",
		"clean":              "@Cleaning=1ALL",
		"deep-clean":         "@Cleaning=2ALL",
		"roller-clean":       "@RollerCleaning=Roller",
		"bottom-plate-clean": "@RollerCleaning=Platen",
	}
	command := commands[action]
	const xml = `<?xml version="1.0" encoding="utf-8" ?>`
	const ivec = `xmlns:ivec="http://www.canon.com/ns/cmd/2008/07/common/"`
	const vcn = `xmlns:vcn="http://www.canon.com/ns/cmd/2008/07/canon/"`
	start := fmt.Sprintf(`%s<cmd %s><ivec:contents><ivec:operation>StartJob</ivec:operation><ivec:param_set servicetype="print"><ivec:jobID>00000001</ivec:jobID><ivec:bidi>0</ivec:bidi></ivec:param_set></ivec:contents></cmd>`, xml, ivec)
	mode := fmt.Sprintf(`%s<cmd %s %s><ivec:contents><ivec:operation>VendorCmd</ivec:operation><ivec:param_set servicetype="print"><vcn:ijoperation>ModeShift</vcn:ijoperation><vcn:ijmode>1</vcn:ijmode><ivec:jobID>00000001</ivec:jobID></ivec:param_set></ivec:contents></cmd>`, xml, ivec, vcn)
	end := fmt.Sprintf(`%s<cmd %s><ivec:contents><ivec:operation>EndJob</ivec:operation><ivec:param_set servicetype="print"><ivec:jobID>00000001</ivec:jobID></ivec:param_set></ivec:contents></cmd>`, xml, ivec)
	bjl := func(lines ...string) []byte {
		body := "BJLSTART\n" + strings.Join(lines, "\n") + "\nBJLEND\n"
		return append([]byte{0x1b, '[', 'K', 0x02, 0x00, 0x00, 0x1f}, []byte(body)...)
	}

	return bytes.Join([][]byte{
		[]byte(start), []byte(mode),
		bjl("ControlMode=Driver", "$JobType=Mnt"),
		bjl("ControlMode=Common", "SetTime="+now.Format("20060102150405")),
		bjl(command), []byte(end),
	}, nil)
}
