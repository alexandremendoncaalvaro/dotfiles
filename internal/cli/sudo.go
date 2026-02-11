package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ale/blueprint/internal/module"
)

// ensureSudo verifica se sudo esta disponivel sem senha.
// Se nao, pede a senha do usuario via terminal (sudo -v).
func ensureSudo() {
	// Tenta sudo non-interactive
	check := exec.Command("sudo", "-n", "true")
	if check.Run() == nil {
		return // sudo ja funciona sem senha
	}

	// Pede senha ao usuario
	fmt.Println("Blueprint precisa de acesso sudo para configurar o sistema.")
	fmt.Println("Digite sua senha (sera pedida apenas uma vez):")
	fmt.Println()

	prompt := exec.Command("sudo", "-v")
	prompt.Stdin = os.Stdin
	prompt.Stdout = os.Stdout
	prompt.Stderr = os.Stderr

	if err := prompt.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Aviso: sudo indisponivel — modulos de sistema podem falhar\n\n")
		return
	}

	fmt.Println()
}

// hasSystemModules retorna true se algum modulo tem a tag "system".
func hasSystemModules(modules []module.Module) bool {
	for _, m := range modules {
		for _, tag := range m.Tags() {
			if tag == "system" {
				return true
			}
		}
	}
	return false
}
