# Canon MG2500 maintenance controls

`/api/admin/printer-maintenance` is an administrator-only API protected by the
normal session/API-key and CSRF middleware. It submits one fixed raw job to the
configured `CANON_MAINTENANCE_PRINTER` queue (default: `Canon_MG2500`). It is
not a generic raw-print endpoint.

## Actions

| Action | Canon BJL command | Purpose |
| --- | --- | --- |
| `nozzle-check` | `@TestPrint=NozzleCheck` | Print a nozzle pattern. |
| `clean` | `@Cleaning=1ALL` | Ordinary head cleaning. |
| `deep-clean` | `@Cleaning=2ALL` | Deep head cleaning; consumes substantially more ink. |
| `roller-clean` | `@RollerCleaning=Roller` | Clean rear paper-feed rollers. |
| `bottom-plate-clean` | `@RollerCleaning=Platen` | Clean the inside/bottom plate. |

Each BJL payload is wrapped in the Canon IVEC StartJob / ModeShift / EndJob
envelope used by the MG2500 driver. The UI requires a separate confirmation for
every state-changing action. It intentionally does not expose power-off, ink
counter reset, or arbitrary BJL commands.

## Operating guidance

- Begin with a nozzle check. Run ordinary cleaning only for missing lines or
  white streaks. Canon recommends deep cleaning only after two ordinary
  cleanings have not fixed the pattern. If deep cleaning does not help, wait
  24 hours before trying it again and then investigate/reseat or replace the
  FINE cartridge.
- Roller cleaning: start with the rear tray empty; when the printer stops the
  roller rotation, load one A4/Letter plain sheet so the machine can eject it.
- Bottom-plate cleaning: use one new A4/Letter plain sheet, folded widthwise
  and unfolded, in the rear tray as specified by Canon.
- Print-head alignment needs a model-specific printed pattern followed by
  placing that pattern on the scanner platen and pressing a physical Start
  button. It is deliberately documented rather than presented as a falsely
  one-click web action.

## Reference and update sources

1. Canon MG2500 [Maintenance procedure](https://ij.manual.canon/ij/webmanual/Manual/M/MG2500%20series/EN/BG/bg-maintenance1100.html), [driver maintenance tab](https://ij.manual.canon/ij/webmanual/PrinterDriver/W/MG2500%20series/1.0/EN/PPG/Dg-Driver_Maintenance.html), [roller cleaning](https://ij.manual.canon/ij/webmanual/Manual/W/MG2500%20series/EN/BG/bg-maintenance2400.html), [bottom-plate cleaning](https://ij.manual.canon/ij/webmanual/Manual/W/MG2500%20series/EN/BG/bg-maintenance2500.html), and [automatic alignment](https://ij.manual.canon/ij/webmanual/PrinterDriver/W/MG2500%20series/1.0/EN/PPG/dg-c_alignment_auto.html) define the supported operations and operator preparation.
2. Canon's public Linux driver source, [willnewton/cnijfilter](https://github.com/willnewton/cnijfilter), `maintenance/src/util_cmd.h` and `util_cmdlist.c`, defines the BJL maintenance vocabulary used for the fixed commands above.
3. [bogdik/mfp-print-scan-server](https://github.com/bogdik/mfp-print-scan-server) is an implementation reference for the IVEC wrapper and CUPS raw-job submission. It is not a runtime dependency. Before adding a new raw action, capture and compare a job emitted by Canon's own driver for the exact printer model; do not infer new command bytes from a related model.
