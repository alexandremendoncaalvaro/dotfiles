// Package devbox cria e provisiona o distrobox de desenvolvimento.
package devbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ale/blueprint/internal/module"
)

// Module implementa a criacao e provisionamento do devbox.
type Module struct {
	// SetupScript e o caminho absoluto para configs/devbox/setup-dev.sh.
	SetupScript string
}

// New cria o modulo devbox.
// setupScript deve ser o caminho absoluto para configs/devbox/setup-dev.sh.
func New(setupScript string) *Module {
	return &Module{SetupScript: setupScript}
}

func (m *Module) Name() string        { return "devbox" }
func (m *Module) Description() string { return "Distrobox de desenvolvimento (criacao + provisionamento)" }
func (m *Module) Tags() []string      { return []string{"system"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container"
	}
	return true, ""
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	output, err := sys.Exec(ctx, "distrobox", "list")
	if err != nil {
		return module.Status{Kind: module.Missing, Message: "distrobox list falhou"}, nil
	}

	if hasContainer(output, "devbox") {
		return module.Status{Kind: module.Installed, Message: "container devbox existe"}, nil
	}

	return module.Status{Kind: module.Missing, Message: "container devbox nao existe"}, nil
}

// hasContainer verifica se um container com o nome exato existe na saida do distrobox list.
// Formato: "ID | NAME | STATUS | IMAGE" — compara o campo NAME sem falso positivo.
func hasContainer(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) >= 2 && strings.TrimSpace(fields[1]) == name {
			return true
		}
	}
	return false
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	reporter.Step(1, 2, "Criando container devbox...")
	homePath := filepath.Join(sys.HomeDir(), ".distrobox", "devbox")
	_, err := sys.Exec(ctx, "distrobox", "create",
		"--name", "devbox",
		"--image", "quay.io/toolbx/ubuntu-toolbox:24.04",
		"--home", homePath,
		"--yes")
	if err != nil {
		// Ignora erro se o container ja existe
		reporter.Warn(fmt.Sprintf("distrobox create retornou erro (container pode ja existir): %v", err))
	} else {
		reporter.Success("Container devbox criado")
	}

	// TODO: considerar ExecStream para dar feedback em tempo real (setup demora minutos)
	reporter.Step(2, 2, "Provisionando devbox...")
	_, err = sys.Exec(ctx, "distrobox", "enter", "devbox", "--", "bash", m.SetupScript)
	if err != nil {
		return fmt.Errorf("erro ao provisionar devbox: %w", err)
	}
	reporter.Success("Devbox provisionado")

	return nil
}
