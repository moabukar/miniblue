package main

import "strings"

// reorderFlagArgs makes Go's standard flag parser behave like "interspersed"
// parsing: flags can appear after positional args.
//
// The standard library flag package stops parsing at the first non-flag.
// That makes ergonomics awkward for commands like:
//
//   specport init keyvault --from arm:...
//
// This helper reorders argv into:
//
//   [flags...] [positionals...]
//
// It is intentionally conservative: it assumes any "-x" / "--x" token that
// does not contain '=' takes the following token as its value unless the
// following token also looks like a flag. This matches how specport's flags
// are used (all are string flags except diff's bool).
func reorderFlagArgs(args []string) []string {
	var flags []string
	var pos []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// If it's a --flag=value form, no extra value.
			if strings.Contains(a, "=") {
				continue
			}
			// If next token exists and is not another flag, treat it as the value.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

