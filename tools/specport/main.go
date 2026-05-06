// specport — Azure spec → miniblue coverage tool.
//
// specport walks a single Azure REST API specification (Swagger 2.0 emitted
// from Azure's TypeSpec sources) for one service and produces a deterministic
// operations checklist. The diff sub-command compares that inventory with
// miniblue's running chi router so PRs can see exactly which Azure operations
// are missing.
//
// Run from the repo root:
//
//	go run ./tools/specport list
//	go run ./tools/specport extract <service>
//	go run ./tools/specport diff <service>
//
// See tools/specport/SKILL.md for the full agent/human runbook.
package main

import (
	"fmt"
	"os"
)

const usageText = `specport — Azure spec → miniblue coverage tool

Usage:
  specport list
      List configured services in tools/specport/specs/.

  specport extract <service> [flags]
      Fetch the Azure REST spec(s) for <service>, parse them, and write
      tools/specport/checklists/<service>.{md,json} with every operation
      marked TODO.

  specport diff <service> [flags]
      Run extract and additionally boot miniblue's chi router to compare
      route coverage. Each operation is marked IMPLEMENTED, MISSING, or
      EXTRA (chi route in miniblue with no spec match).

Flags:
  -spec-dir   Override the directory holding service config files.
              Default: tools/specport/specs
  -out-dir    Override the directory where checklists are written.
              Default: tools/specport/checklists

Run from the repo root.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "list":
		err = cmdList(os.Args[2:])
	case "extract":
		err = cmdExtract(os.Args[2:])
	case "diff":
		err = cmdDiff(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usageText)
	default:
		fmt.Fprintf(os.Stderr, "specport: unknown command %q\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "specport: %v\n", err)
		os.Exit(1)
	}
}
