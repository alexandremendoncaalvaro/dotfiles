// Package starship configura o prompt Starship.
// Instala o binario, cria symlink da config e adiciona init ao shell.
package starship

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ale/dotfiles/internal/module"
)

// Module implementa a configuracao do Starship.
type Module struct {
	// ConfigSource e o caminho absoluto do starship.toml no repo.
	ConfigSource string
}

// New cria o modulo Starship.
// configSource deve ser o caminho absoluto para configs/starship.toml.
func New(configSource string) *Module {
	return &Module{ConfigSource: configSource}
}

func (m *Module) Name() string        { return "starship" }
func (m *Module) Description() string { return "Prompt Starship (instalacao + config + shell init)" }
func (m *Module) Tags() []string      { return []string{"shell"} }

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	hasCmd := sys.CommandExists("starship")
	configPath := filepath.Join(sys.HomeDir(), ".config", "starship.toml")
	hasConfig := sys.FileExists(configPath)

	bashrc := filepath.Join(sys.HomeDir(), ".bashrc")
	hasBashInit := false
	if data, err := sys.ReadFile(bashrc); err == nil {
		hasBashInit = strings.Contains(string(data), `eval "$(starship init bash)"`)
	}

	if hasCmd && hasConfig && hasBashInit {
		return module.Status{Kind: module.Installed, Message: "Starship instalado e configurado"}, nil
	}

	if hasCmd || hasConfig {
		msg := "Starship parcialmente configurado:"
		if !hasCmd {
			msg += " binario ausente;"
		}
		if !hasConfig {
			msg += " config ausente;"
		}
		if !hasBashInit {
			msg += " init no bashrc ausente;"
		}
		return module.Status{Kind: module.Partial, Message: msg}, nil
	}

	return module.Status{Kind: module.Missing, Message: "Starship nao instalado"}, nil
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	// 1. Instalar Starship se ausente
	if !sys.CommandExists("starship") {
		reporter.Step(1, 4, "Instalando Starship...")
		_, err := sys.Exec(ctx, "sh", "-c", `curl -sS https://starship.rs/install.sh | sh -s -- -y`)
		if err != nil {
			return fmt.Errorf("erro ao instalar starship (verifique sua conexao com a internet): %w", err)
		}
		reporter.Success("Starship instalado")
	} else {
		reporter.Step(1, 4, "Starship ja instalado")
	}

	// 2. Criar symlink da config
	reporter.Step(2, 4, "Configurando starship.toml...")
	configDir := filepath.Join(sys.HomeDir(), ".config")
	if err := sys.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("erro ao criar diretorio .config: %w", err)
	}

	configDest := filepath.Join(configDir, "starship.toml")
	if err := sys.Symlink(m.ConfigSource, configDest); err != nil {
		return fmt.Errorf("erro ao criar symlink: %w", err)
	}
	reporter.Success("Symlink criado: starship.toml")

	// 3. Adicionar init ao .bashrc
	reporter.Step(3, 4, "Configurando .bashrc...")
	bashrc := filepath.Join(sys.HomeDir(), ".bashrc")
	initLine := `eval "$(starship init bash)"`
	added, err := sys.AppendToFileIfMissing(bashrc, initLine)
	if err != nil {
		return fmt.Errorf("erro ao configurar .bashrc: %w", err)
	}
	if added {
		reporter.Success("Starship adicionado ao .bashrc")
	} else {
		reporter.Info("Starship ja esta no .bashrc")
	}

	// 4. Adicionar init ao .zshrc (se existir)
	reporter.Step(4, 4, "Verificando .zshrc...")
	zshrc := filepath.Join(sys.HomeDir(), ".zshrc")
	if sys.FileExists(zshrc) {
		zshInitLine := `eval "$(starship init zsh)"`
		added, err := sys.AppendToFileIfMissing(zshrc, zshInitLine)
		if err != nil {
			return fmt.Errorf("erro ao configurar .zshrc: %w", err)
		}
		if added {
			reporter.Success("Starship adicionado ao .zshrc")
		} else {
			reporter.Info("Starship ja esta no .zshrc")
		}
	} else {
		reporter.Info("Sem .zshrc, pulando")
	}

	return nil
}


