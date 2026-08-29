package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/cyprienbrisset/kanjo/internal/version"
)

type doctorReport struct {
	Tool         string            `json:"tool"`
	RulesVersion string            `json:"rulesVersion"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	GoVersion    string            `json:"goVersion"`
	Capabilities map[string]capab  `json:"capabilities"`
	Paths        map[string]string `json:"paths"`
}

type capab struct {
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Note      string `json:"note,omitempty"`
}

func runDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	format := fs.String("format", "", "format de sortie : table|json")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	rep := doctorReport{
		Tool:         version.Tool,
		RulesVersion: version.Rules,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		Capabilities: map[string]capab{},
		Paths:        map[string]string{},
	}
	// Outils externes optionnels (jamais requis, §9.2, §8.4).
	for _, tool := range []struct{ name, bin, note string }{
		{"ghostscript", "gs", "conversion PDF/A optionnelle (embed --gs)"},
		{"verapdf", "verapdf", "validation PDF/A optionnelle"},
		{"java", "java", "validateurs externes optionnels (cross-check)"},
	} {
		p, err := exec.LookPath(tool.bin)
		rep.Capabilities[tool.name] = capab{Available: err == nil, Path: p, Note: tool.note}
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		rep.Paths["config"] = cfg + "/kanjo"
	}
	if cache, err := os.UserCacheDir(); err == nil {
		rep.Paths["cache"] = cache + "/kanjo"
	}

	if outputFormat(*format) == "json" {
		printJSON(rep)
		return ExitOK
	}
	fmt.Fprintf(os.Stdout, "kanjo %s · règles %s\n", rep.Tool, rep.RulesVersion)
	fmt.Fprintf(os.Stdout, "plateforme   %s/%s (%s)\n", rep.OS, rep.Arch, rep.GoVersion)
	fmt.Fprintln(os.Stdout, "capacités optionnelles :")
	for _, name := range []string{"ghostscript", "verapdf", "java"} {
		c := rep.Capabilities[name]
		mark := "absent"
		if c.Available {
			mark = "présent (" + c.Path + ")"
		}
		fmt.Fprintf(os.Stdout, "  %-12s %s — %s\n", name, mark, c.Note)
	}
	for _, k := range []string{"config", "cache"} {
		if v, ok := rep.Paths[k]; ok {
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", k, v)
		}
	}
	return ExitOK
}
