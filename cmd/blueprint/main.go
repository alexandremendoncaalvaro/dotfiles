// Entry point do blueprint.
// Registra todos os modulos e inicia a CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ale/blueprint/internal/cli"
	"github.com/ale/blueprint/internal/module"
	"github.com/ale/blueprint/internal/modules/bluefin_update"
	"github.com/ale/blueprint/internal/modules/cedilla"
	"github.com/ale/blueprint/internal/modules/clipboard_indicator"
	"github.com/ale/blueprint/internal/modules/gnome_focus"
	"github.com/ale/blueprint/internal/modules/gnome_forge"
	"github.com/ale/blueprint/internal/modules/passwordless"
	"github.com/ale/blueprint/internal/modules/starship"
	"github.com/ale/blueprint/internal/system"
)

func main() {
	// Descobre o diretorio do repo (onde o binario esta ou diretorio de trabalho)
	repoDir := discoverRepoDir()

	// Cria o sistema real
	sys := system.NewReal()

	// Registra modulos
	reg := module.NewRegistry()

	configSource := filepath.Join(repoDir, "configs", "starship.toml")
	must(reg.Register(starship.New(configSource)))
	must(reg.Register(cedilla.New()))
	must(reg.Register(gnome_forge.New()))
	must(reg.Register(clipboard_indicator.New()))

	focusExtSource := filepath.Join(repoDir, "configs", "gnome-extensions", "focus-mode@blueprint")
	must(reg.Register(gnome_focus.New(focusExtSource)))

	must(reg.Register(bluefin_update.New()))
	must(reg.Register(passwordless.New()))

	// Configura a app
	app := &cli.App{
		Registry:  reg,
		System:    sys,
		Options:   &cli.Options{},
		ConfigDir: filepath.Join(repoDir, "configs"),
	}

	// Executa
	cmd := cli.NewRootCmd(app)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}

// discoverRepoDir tenta encontrar o diretorio raiz do repositorio.
// Prioridade: BLUEPRINT_DIR env > diretorio do executavel > ~/blueprint > diretorio atual.
func discoverRepoDir() string {
	// 1. Variavel de ambiente
	if dir := os.Getenv("BLUEPRINT_DIR"); dir != "" {
		return dir
	}

	// 2. Diretorio do executavel
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		// Se o binario esta em bin/, o repo e o pai
		if filepath.Base(exeDir) == "bin" {
			parent := filepath.Dir(exeDir)
			if isRepoDir(parent) {
				return parent
			}
		}
		if isRepoDir(exeDir) {
			return exeDir
		}
	}

	// 3. ~/blueprint
	home, _ := os.UserHomeDir()
	if home != "" {
		blueprintDir := filepath.Join(home, "blueprint")
		if isRepoDir(blueprintDir) {
			return blueprintDir
		}
	}

	// 4. Diretorio atual
	cwd, _ := os.Getwd()
	return cwd
}

// isRepoDir verifica se o diretorio parece ser a raiz do repo.
func isRepoDir(dir string) bool {
	// Verifica se tem go.mod (indicador do projeto)
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro fatal: %v\n", err)
		os.Exit(1)
	}
}
