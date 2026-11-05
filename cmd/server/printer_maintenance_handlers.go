package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

const (
	canonMaintenanceDefaultPrinter = "Canon_MG2500"
	canonMaintenanceTimeout        = 30 * time.Second
	// Increment when the fixed Canon maintenance command vocabulary changes.
	// Returning it lets operators confirm which reviewed protocol revision is live.
	canonMaintenanceProtocolRevision = "2026-11-04.1"
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
	case "test-page":
		title = "Canon print test page"
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
	ctx, cancel := context.WithTimeout(r.Context(), canonMaintenanceTimeout)
	defer cancel()
	var cmd *exec.Cmd
	var cleanup func()
	if action == "test-page" {
		path, pageCleanup, err := generateMaintenanceTestPage()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate test page")
			return
		}
		cleanup = pageCleanup
		if os.Getenv("CANON_RASTERIZE_DOCUMENTS") == "true" {
			rasterPath, rasterCleanup, err := rasterizePDFForCanon(ctx, path, "portrait", "A4")
			if err != nil {
				cleanup()
				writeJSONError(w, http.StatusBadGateway, "failed to rasterize test page")
				return
			}
			previousCleanup := cleanup
			cleanup = func() { rasterCleanup(); previousCleanup() }
			path = rasterPath
		}
		defer cleanup()
		// Deliberately do not use -o raw: this is an ordinary CUPS print job so
		// the selected Canon driver, paper handling and color pipeline are tested.
		cmd = exec.CommandContext(ctx, "lp", "-d", printer, "-t", title, "-o", "media=A4", path)
	} else {
		payload := canonMaintenancePayload(action, time.Now())
		cmd = exec.CommandContext(ctx, "lp", "-d", printer, "-o", "raw", "-t", title, "-")
		cmd.Stdin = bytes.NewReader(payload)
	}
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
		"ok":               "true",
		"action":           action,
		"printer":          printer,
		"job":              job,
		"protocolRevision": canonMaintenanceProtocolRevision,
	})
}

// generateMaintenanceTestPage creates one A4 page exercising text, solid
// black, grayscale, CMYK-like color patches, and thin rules. It is intentionally
// generated locally so the maintenance endpoint never fetches user-controlled
// content or exposes a generic raw-print primitive.
func generateMaintenanceTestPage() (string, func(), error) {
	dir, err := os.MkdirTemp("", "printer-test-page-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "printer-test-page.pdf")
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Printer Test Page", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(0, 10, "Printer Test Page", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, "Verify sharp text, smooth grayscale, solid black, cyan, magenta, yellow and the fine lines below. Generated locally by printer maintenance.", "", "L", false)
	pdf.Ln(4)

	patches := []struct {
		r, g, b int
		label   string
	}{
		{0, 0, 0, "Black"}, {0, 174, 239, "Cyan"}, {236, 0, 140, "Magenta"}, {255, 242, 0, "Yellow"},
	}
	for _, p := range patches {
		pdf.SetFillColor(p.r, p.g, p.b)
		pdf.Rect(pdf.GetX(), pdf.GetY(), 42, 22, "F")
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(42, 22, p.label, "", 0, "C", false, 0, "")
	}
	pdf.Ln(28)
	for i := 0; i < 11; i++ {
		v := 255 - i*25
		pdf.SetFillColor(v, v, v)
		pdf.Rect(10+float64(i)*17, pdf.GetY(), 17, 16, "F")
	}
	pdf.Ln(23)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)
	pdf.Cell(0, 5, "Grayscale: 100% black to 0% black")
	pdf.Ln(10)
	for i := 0; i < 12; i++ {
		pdf.SetLineWidth(0.1 + float64(i)*0.05)
		pdf.Line(15, pdf.GetY()+float64(i)*4, 195, pdf.GetY()+float64(i)*4)
	}
	pdf.SetY(pdf.GetY() + 55)
	pdf.SetLineWidth(0.2)
	pdf.SetFont("Helvetica", "", 7)
	pdf.MultiCell(0, 4, "ABCDEFGHIJKLMNOPQRSTUVWXYZ  abcdefghijklmnopqrstuvwxyz  0123456789\nThe quick brown fox jumps over the lazy dog.", "", "L", false)
	pdf.SetY(282)
	pdf.SetTextColor(90, 90, 90)
	pdf.Cell(0, 5, "A4 · one page · maintenance test only")
	if err := pdf.OutputFileAndClose(path); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
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
