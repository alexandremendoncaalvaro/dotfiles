package devcontainers

import (
	"context"
	"fmt"

	"github.com/ale/blueprint/internal/module"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "devcontainers" }
func (m *Module) Description() string { return "Dev Containers (dev mode + podman-docker)" }
func (m *Module) Tags() []string      { return []string{"system"} }

func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container"
	}
	return true, ""
}

func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	devMode := checkDevMode(ctx, sys)
	hasDockerCE := checkRPM(ctx, sys, "docker-ce")
	hasPodmanDocker := checkRPM(ctx, sys, "podman-docker")

	switch {
	case devMode && !hasDockerCE && hasPodmanDocker:
		return module.Status{Kind: module.Installed, Message: "Dev mode ativo, podman-docker instalado"}, nil
	case !devMode && hasDockerCE && !hasPodmanDocker:
		return module.Status{Kind: module.Missing, Message: "Dev mode inativo, docker-ce presente, podman-docker ausente"}, nil
	default:
		return module.Status{Kind: module.Partial, Message: "Configuracao parcial"}, nil
	}
}

func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	reporter.Step(1, 3, "Habilitando dev mode...")
	if _, err := sys.Exec(ctx, "ujust", "devmode-enable"); err != nil {
		return fmt.Errorf("erro ao habilitar dev mode: %w", err)
	}
	reporter.Success("Dev mode habilitado")

	reporter.Step(2, 3, "Removendo Docker CE...")
	_, err := sys.Exec(ctx, "rpm-ostree", "override", "remove",
		"docker-ce", "docker-ce-cli", "docker-ce-rootless-extras")
	if err != nil {
		reporter.Warn(fmt.Sprintf("Remocao do Docker CE falhou (pode ja estar removido): %v", err))
	}

	reporter.Step(3, 3, "Instalando podman-docker...")
	if _, err := sys.Exec(ctx, "rpm-ostree", "install", "podman-docker"); err != nil {
		return fmt.Errorf("erro ao instalar podman-docker: %w", err)
	}
	reporter.Success("podman-docker instalado")

	reporter.Warn("Reboot necessario para aplicar as alteracoes")
	return nil
}

func checkDevMode(ctx context.Context, sys module.System) bool {
	_, err := sys.Exec(ctx, "ujust", "devmode")
	return err == nil
}

func checkRPM(ctx context.Context, sys module.System, pkg string) bool {
	_, err := sys.Exec(ctx, "rpm", "-q", pkg)
	return err == nil
}
