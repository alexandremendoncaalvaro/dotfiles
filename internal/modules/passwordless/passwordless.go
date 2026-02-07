// Package passwordless configura sudo sem senha e login automatico no GDM.
package passwordless

import (
	"context"
	"fmt"
	"strings"

	"github.com/ale/blueprint/internal/module"
)

// Module implementa sudo sem senha e login automatico no GDM.
type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string        { return "passwordless" }
func (m *Module) Description() string { return "Sudo sem senha e login automatico no GDM" }
func (m *Module) Tags() []string      { return []string{"system"} }

// ShouldRun retorna false dentro de containers.
func (m *Module) ShouldRun(_ context.Context, sys module.System) (bool, string) {
	if sys.IsContainer() {
		return false, "dentro de container (configuracao de sistema)"
	}
	return true, ""
}

// Check verifica se sudo sem senha e login automatico estao configurados.
func (m *Module) Check(ctx context.Context, sys module.System) (module.Status, error) {
	sudoOK := checkSudo(ctx, sys)
	gdmOK := checkGDM(sys)

	switch {
	case sudoOK && gdmOK:
		return module.Status{Kind: module.Installed, Message: "Sudo sem senha e login automatico configurados"}, nil
	case !sudoOK && !gdmOK:
		return module.Status{Kind: module.Missing, Message: "Sudo com senha e login manual"}, nil
	default:
		var msg string
		if sudoOK {
			msg = "Sudo sem senha OK, login automatico ausente"
		} else {
			msg = "Login automatico OK, sudo com senha"
		}
		return module.Status{Kind: module.Partial, Message: msg}, nil
	}
}

// checkSudo testa se sudo funciona sem senha.
func checkSudo(ctx context.Context, sys module.System) bool {
	_, err := sys.Exec(ctx, "sudo", "-n", "true")
	return err == nil
}

// checkGDM verifica se o login automatico esta configurado no GDM.
func checkGDM(sys module.System) bool {
	const gdmConf = "/etc/gdm/custom.conf"

	data, err := sys.ReadFile(gdmConf)
	if err != nil {
		return false
	}

	user := sys.Env("USER")
	if user == "" {
		return false
	}

	content := string(data)
	return strings.Contains(content, "AutomaticLoginEnable=True") &&
		strings.Contains(content, "AutomaticLogin="+user)
}

// Apply configura sudo sem senha e login automatico no GDM.
func (m *Module) Apply(ctx context.Context, sys module.System, reporter module.Reporter) error {
	user := sys.Env("USER")
	if user == "" {
		return fmt.Errorf("variavel USER nao definida")
	}

	cacheDir := sys.HomeDir() + "/.cache"

	// Step 1 — Sudo sem senha
	reporter.Step(1, 2, "Configurando sudo sem senha...")

	sudoersContent := user + " ALL=(ALL) NOPASSWD: ALL\n"
	tmpSudoers := cacheDir + "/blueprint-nopasswd"
	sudoersTarget := "/etc/sudoers.d/nopasswd-" + user

	if err := sys.WriteFile(tmpSudoers, []byte(sudoersContent), 0o644); err != nil {
		return fmt.Errorf("erro ao escrever arquivo temporario de sudoers: %w", err)
	}

	if _, err := sys.Exec(ctx, "sudo", "visudo", "-c", "-f", tmpSudoers); err != nil {
		return fmt.Errorf("validacao do sudoers falhou: %w", err)
	}

	if _, err := sys.Exec(ctx, "sudo", "cp", tmpSudoers, sudoersTarget); err != nil {
		return fmt.Errorf("erro ao copiar sudoers: %w", err)
	}

	if _, err := sys.Exec(ctx, "sudo", "chmod", "0440", sudoersTarget); err != nil {
		return fmt.Errorf("erro ao ajustar permissoes do sudoers: %w", err)
	}

	reporter.Success("Sudo sem senha configurado")

	// Step 2 — Login automatico no GDM
	reporter.Step(2, 2, "Configurando login automatico no GDM...")

	const gdmConf = "/etc/gdm/custom.conf"
	gdmContent, err := sys.ReadFile(gdmConf)
	if err != nil {
		return fmt.Errorf("erro ao ler %s: %w", gdmConf, err)
	}

	newContent := setGDMAutoLogin(string(gdmContent), user)
	tmpGDM := cacheDir + "/blueprint-gdm-custom.conf"

	if err := sys.WriteFile(tmpGDM, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("erro ao escrever arquivo temporario do GDM: %w", err)
	}

	if _, err := sys.Exec(ctx, "sudo", "cp", tmpGDM, gdmConf); err != nil {
		return fmt.Errorf("erro ao copiar configuracao do GDM: %w", err)
	}

	reporter.Success("Login automatico configurado")

	return nil
}

// setGDMAutoLogin adiciona/atualiza as chaves de login automatico na secao [daemon].
func setGDMAutoLogin(content, user string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inDaemon := false
	setEnable := false
	setUser := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detecta secoes
		if strings.HasPrefix(trimmed, "[") {
			// Se estamos saindo de [daemon] sem ter adicionado as chaves, adiciona
			if inDaemon {
				if !setEnable {
					result = append(result, "AutomaticLoginEnable=True")
				}
				if !setUser {
					result = append(result, "AutomaticLogin="+user)
				}
			}
			inDaemon = strings.EqualFold(trimmed, "[daemon]")
		}

		// Atualiza chaves existentes na secao [daemon]
		if inDaemon {
			if strings.HasPrefix(trimmed, "AutomaticLoginEnable=") || strings.HasPrefix(trimmed, "AutomaticLoginEnable =") {
				result = append(result, "AutomaticLoginEnable=True")
				setEnable = true
				continue
			}
			if strings.HasPrefix(trimmed, "AutomaticLogin=") || strings.HasPrefix(trimmed, "AutomaticLogin =") {
				// Cuidado para nao capturar AutomaticLoginEnable
				if !strings.HasPrefix(trimmed, "AutomaticLoginEnable") {
					result = append(result, "AutomaticLogin="+user)
					setUser = true
					continue
				}
			}
		}

		result = append(result, line)
	}

	// Se [daemon] era a ultima secao e nao adicionamos as chaves
	if inDaemon {
		if !setEnable {
			result = append(result, "AutomaticLoginEnable=True")
		}
		if !setUser {
			result = append(result, "AutomaticLogin="+user)
		}
	}

	return strings.Join(result, "\n")
}
