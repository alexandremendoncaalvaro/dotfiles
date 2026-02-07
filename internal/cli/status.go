package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ale/dotfiles/internal/module"
	"github.com/ale/dotfiles/internal/orchestrator"
	"github.com/ale/dotfiles/internal/profile"
	"github.com/ale/dotfiles/internal/tui"
	"github.com/ale/dotfiles/internal/version"
	"github.com/spf13/cobra"
)

// maxWidth for module name column alignment.
const maxNameWidth = 20

// ANSI colors for terminal output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func newStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Mostrar estado detalhado dos modulos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			sys := app.System

			var prof profile.Profile
			autoDetected := false
			if app.Options.Profile == "auto" {
				prof = profile.Detect(sys)
				autoDetected = true
			} else {
				var err error
				prof, err = profile.ByName(app.Options.Profile)
				if err != nil {
					return err
				}
			}
			modules := profile.Resolve(prof, app.Registry)

			reporter := tui.NewHeadlessReporter()
			orch := orchestrator.New(sys, reporter)
			results := orch.CheckAll(ctx, modules)

			// ── Header ──────────────────────────────
			hostname, _ := os.Hostname()
			fmt.Printf("\n%s%s dotfiles status%s", colorBold, colorCyan, colorReset)
			if version.Version != "" {
				fmt.Printf("  %s%s%s", colorDim, version.Version, colorReset)
			}
			fmt.Println()
			fmt.Printf("%s─────────────────────────────────────%s\n", colorDim, colorReset)

			fmt.Printf("  Perfil:    %s%s%s", colorBold, prof.Name, colorReset)
			if autoDetected {
				fmt.Printf("  %s(auto-detectado)%s", colorDim, colorReset)
			}
			fmt.Println()
			if hostname != "" {
				fmt.Printf("  Host:      %s\n", hostname)
			}
			if sys.IsContainer() {
				fmt.Printf("  Ambiente:  %scontainer%s\n", colorYellow, colorReset)
			} else {
				session := sys.Env("XDG_SESSION_TYPE")
				if session != "" {
					fmt.Printf("  Sessão:    %s\n", session)
				}
			}
			fmt.Println()

			// ── Modules ──────────────────────────────
			installed, total := 0, len(results)
			for _, r := range results {
				if r.Status.Kind == module.Installed {
					installed++
				}
			}
			fmt.Printf("  %s%d/%d módulos OK%s\n\n", colorBold, installed, total, colorReset)

			for _, r := range results {
				icon, color := statusStyle(r.Status.Kind)
				fmt.Printf("  %s%s%s  %s%-*s%s  %s%s%s\n",
					color, icon, colorReset,
					colorBold, maxNameWidth, r.Module.Name(), colorReset,
					colorDim, r.Status.Message, colorReset)
			}
			fmt.Println()

			// ── Skipped modules ──────────────────────
			allModules := app.Registry.All()
			resolved := make(map[string]bool)
			for _, m := range modules {
				resolved[m.Name()] = true
			}
			var skipped []string
			for _, m := range allModules {
				if !resolved[m.Name()] {
					skipped = append(skipped, m.Name())
				}
			}
			if len(skipped) > 0 {
				fmt.Printf("  %sFora do perfil:%s %s\n\n", colorDim, colorReset, strings.Join(skipped, ", "))
			}

			return nil
		},
	}
}

func statusStyle(kind module.StatusKind) (string, string) {
	switch kind {
	case module.Installed:
		return "✔", colorGreen
	case module.Missing:
		return "✘", colorRed
	case module.Partial:
		return "◐", colorYellow
	case module.Skipped:
		return "⊘", colorDim
	default:
		return "?", colorReset
	}
}
