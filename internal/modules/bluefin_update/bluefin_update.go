// Package bluefin_update executa atualizacoes do sistema Bluefin.
// Roda rpm-ostree, flatpak, fwupd e distrobox conforme disponibilidade.
package bluefin_update

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/dotfiles/internal/module"
)

// step define um passo de atualizacao.
type step struct {
	name     string   // Nome para exibicao
	cmd      string   // Comando principal
	args     []string // Argumentos
	optional bool     // Se true, pula se o comando nao existir
}

// Module implementa a atualizacao do sistema Bluefin.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "bluefin-update" }
func (m *Module) Description() string {
	return "Atualizar sistema Bluefin (rpm-ostree, Flatpak, fwupd, Distrobox)"
}
func (m *Module) Tags() []string { return []string{"system"} }

// ShouldRun retorna false dentro de containers.
func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container (atualizacao de sistema)"
	}
	return true, ""
}

// Check verifica se ha atualizacoes pendentes no sistema.
// Consulta rpm-ostree e flatpak para determinar o estado.
func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	rpmPending := false
	flatpakPending := false

	// rpm-ostree upgrade --check retorna exit 0 se ha updates, exit 77 se nao ha.
	// Qualquer outro erro (ex: rede) e ignorado — assumimos "sem updates".
	if sys.CommandExists("rpm-ostree") {
		_, err := sys.Exec(ctx, "rpm-ostree", "upgrade", "--check")
		if err == nil {
			// exit 0 = updates disponiveis
			rpmPending = true
		}
		// exit != 0 (incluindo 77) = sem updates ou erro
	}

	// flatpak remote-ls --updates lista pacotes com atualizacao pendente.
	// Se a saida nao esta vazia, ha updates.
	if sys.CommandExists("flatpak") {
		out, err := sys.Exec(ctx, "flatpak", "remote-ls", "--updates")
		if err == nil && strings.TrimSpace(out) != "" {
			flatpakPending = true
		}
	}

	switch {
	case rpmPending && flatpakPending:
		return module.Status{Kind: module.Missing, Message: "Atualizacoes disponiveis (sistema e Flatpak)"}, nil
	case rpmPending:
		return module.Status{Kind: module.Missing, Message: "Atualizacao do sistema disponivel (rpm-ostree)"}, nil
	case flatpakPending:
		return module.Status{Kind: module.Missing, Message: "Atualizacoes Flatpak disponiveis"}, nil
	default:
		return module.Status{Kind: module.Installed, Message: "Sistema atualizado"}, nil
	}
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	steps := []step{
		{
			name: "rpm-ostree (base do sistema)",
			cmd:  "rpm-ostree",
			args: []string{"upgrade"},
		},
		{
			name: "Flatpak (aplicativos)",
			cmd:  "flatpak",
			args: []string{"update", "-y"},
		},
		{
			name:     "fwupd (firmware)",
			cmd:      "fwupdmgr",
			args:     []string{"refresh"},
			optional: true,
		},
		{
			name:     "fwupd (atualizar firmware)",
			cmd:      "fwupdmgr",
			args:     []string{"update"},
			optional: true,
		},
		{
			name:     "Distrobox (containers)",
			cmd:      "distrobox",
			args:     []string{"upgrade", "--all"},
			optional: true,
		},
	}

	total := len(steps)
	for i, s := range steps {
		reporter.Step(i+1, total, fmt.Sprintf("Atualizando %s...", s.name))

		if !sys.CommandExists(s.cmd) {
			if s.optional {
				reporter.Info(fmt.Sprintf("%s nao encontrado, pulando", s.cmd))
				continue
			}
			return fmt.Errorf("comando obrigatorio nao encontrado: %s — voce esta rodando em um sistema Bluefin?", s.cmd)
		}

		err := sys.ExecStream(ctx, func(line string) {
			reporter.Info(line)
		}, s.cmd, s.args...)

		if err != nil {
			if s.optional {
				reporter.Warn(fmt.Sprintf("%s falhou (opcional): %v", s.name, err))
				continue
			}
			return fmt.Errorf("%s falhou: %w", s.name, err)
		}

		reporter.Success(fmt.Sprintf("%s concluido", s.name))
	}

	return nil
}
