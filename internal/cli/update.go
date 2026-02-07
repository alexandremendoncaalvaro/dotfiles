package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUpdateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Atualizar dotfiles (git pull + rebuild)",
		Long:  "Puxa a versão mais recente do repositório e recompila o binário.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			sys := app.System

			// Descobrir raiz do repo (diretório do executável → ../.. ou via git)
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("nao foi possivel determinar o caminho do executavel: %w", err)
			}
			repoDir := filepath.Dir(filepath.Dir(exe)) // bin/dotfiles → repo root

			// 1. git pull
			fmt.Printf("  Atualizando repositório em %s...\n", repoDir)
			out, err := sys.Exec(ctx, "git", "-C", repoDir, "pull", "--ff-only")
			if err != nil {
				return fmt.Errorf("git pull falhou (verifique se o repo nao tem mudancas locais): %w", err)
			}
			fmt.Printf("  %s\n", out)

			if out == "Already up to date." || out == "Already up to date.\n" {
				fmt.Println("  ✔ Já está na versão mais recente")
				return nil
			}

			// 2. Rebuild
			fmt.Println("  Recompilando...")
			if _, err := sys.Exec(ctx, "make", "-C", repoDir, "build"); err != nil {
				return fmt.Errorf("build falhou: %w", err)
			}
			fmt.Println("  ✔ Binário atualizado")

			fmt.Println()
			fmt.Println("  Rode 'dotfiles apply' para aplicar as mudanças.")
			return nil
		},
	}
}
