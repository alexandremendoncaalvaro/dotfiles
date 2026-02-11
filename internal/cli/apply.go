package cli

import (
	"fmt"

	"github.com/ale/blueprint/internal/orchestrator"
	"github.com/ale/blueprint/internal/profile"
	"github.com/ale/blueprint/internal/system"
	"github.com/ale/blueprint/internal/tui"
	"github.com/spf13/cobra"
)

func newApplyCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [perfil]",
		Short: "Aplicar configuracoes do perfil selecionado",
		Long:  "Aplica todos os modulos do perfil. Sem argumentos, detecta o perfil automaticamente.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Perfil via argumento tem prioridade
			if len(args) > 0 {
				app.Options.Profile = args[0]
			}

			// Resolve perfil: auto-detecta ou usa o explicito
			autoDetected := false
			var prof profile.Profile
			if app.Options.Profile == "auto" {
				prof = profile.Detect(app.System)
				autoDetected = true
			} else {
				var err error
				prof, err = profile.ByName(app.Options.Profile)
				if err != nil {
					return err
				}
			}

			modules := profile.Resolve(prof, app.Registry)

			if len(modules) == 0 {
				fmt.Println("Nenhum modulo encontrado para o perfil:", prof.Name)
				return nil
			}

			// Sudo interativo: pede senha antes de iniciar TUI/headless
			if !app.Options.DryRun && !app.System.IsContainer() && hasSystemModules(modules) {
				ensureSudo()
			}

			// Configura dry-run se necessario
			sys := app.System
			if app.Options.DryRun {
				sys = system.NewDryRun(app.System, func(msg string) {
					fmt.Println(msg)
				})
			}

			mode := DetectMode(app.Options.Headless)

			if mode == Interactive {
				return tui.Run(app.Registry, sys, prof, autoDetected)
			}

			// Modo headless
			reporter := tui.NewHeadlessReporter()
			orch := orchestrator.New(sys, reporter)

			fmt.Printf("Aplicando perfil: %s (%d modulos)\n", prof.Name, len(modules))
			fmt.Println()

			results := orch.Run(cmd.Context(), modules)

			// Resumo
			fmt.Println()
			fmt.Println("=== Resumo ===")
			var errs int
			for _, r := range results {
				icon := "OK"
				if r.Skipped {
					icon = "SKIP"
				} else if r.Err != nil {
					icon = "ERRO"
					errs++
				} else if r.Applied {
					icon = "APLICADO"
				}
				fmt.Printf("  [%s] %s\n", icon, r.Module.Name())
			}

			// Notas pos-apply (instrucoes importantes para o usuario)
			var allNotes []string
			for _, r := range results {
				allNotes = append(allNotes, r.Notes...)
			}
			if len(allNotes) > 0 {
				fmt.Println()
				fmt.Println("=== Proximos passos ===")
				for _, note := range allNotes {
					fmt.Printf("  %s\n", note)
				}
			}

			if errs > 0 {
				return fmt.Errorf("%d modulo(s) com erro", errs)
			}

			fmt.Println()
			fmt.Println("Concluido!")
			return nil
		},
	}

	return cmd
}
