package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/cyprienbrisset/kanjo/internal/fsatomic"
	"github.com/cyprienbrisset/kanjo/pkg/read"
	"github.com/cyprienbrisset/kanjo/pkg/repair"
	"github.com/cyprienbrisset/kanjo/pkg/write"
)

func runRepair(args []string) int {
	fs := flag.NewFlagSet("repair", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "lister les corrections sans les appliquer")
	backup := fs.Bool("backup", true, "conserver l'original (.bak)")
	format := fs.String("format", "", "format de sortie (défaut : identique à l'entrée)")
	var fixes stringSlice
	fs.Var(&fixes, "fixes", "corrections à appliquer (défaut : toutes les sûres)")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(inputs) == 0 {
		errf("repair : aucun fichier d'entrée")
		return ExitUsage
	}

	var opts repair.Options
	for _, f := range fixes {
		opts.Fixes = append(opts.Fixes, repair.Fix(f))
	}

	worst := ExitOK
	for _, in := range inputs {
		data, err := readInput(in)
		if err != nil {
			errf("%v", err)
			worst = maxExit(worst, ExitUnreadable)
			continue
		}
		rd, err := read.ReadBytes(data, in)
		if err != nil {
			errf("%v", err)
			worst = maxExit(worst, ExitUnreadable)
			continue
		}
		changes := repair.Repair(rd.Doc, opts)
		if len(changes) == 0 {
			fmt.Fprintf(os.Stdout, "✓ %s : aucune correction nécessaire\n", in)
			continue
		}
		fmt.Fprintf(os.Stdout, "⚠ %s : %d correction(s)\n", in, len(changes))
		for _, c := range changes {
			fmt.Fprintf(os.Stdout, "    %s : %s → %s (%s)\n", c.Path, c.Before, c.After, c.Fix)
		}
		if *dryRun {
			continue
		}

		target := *format
		if target == "" {
			target = formatToTarget(rd.Format)
		}
		outData, err := write.WriteBytes(target, rd.Doc, write.Options{Profile: write.ProfileEN16931, Indent: true})
		if err != nil {
			errf("repair %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
			continue
		}
		if *backup && in != "-" {
			if err := fsatomic.WriteFile(in+".bak", data, 0o644); err != nil {
				errf("repair %s : sauvegarde impossible : %v", in, err)
			}
		}
		outPath := in
		if in == "-" {
			os.Stdout.Write(outData)
			continue
		}
		// Ne jamais écraser un PDF Factur-X avec du XML : sortie sidecar.
		if rd.Format == read.FormatFacturX {
			outPath = in + ".repaired.xml"
		}
		if err := fsatomic.WriteFile(outPath, outData, 0o644); err != nil {
			errf("repair %s : %v", in, err)
			worst = maxExit(worst, ExitInternal)
		}
	}
	return worst
}
