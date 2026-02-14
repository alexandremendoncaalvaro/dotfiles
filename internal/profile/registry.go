package profile

import (
	"fmt"
	"strings"
)

// Perfis built-in disponiveis.
var (
	Full = Profile{
		Name:        "full",
		Description: "Desktop Bluefin completo (shell + desktop + sistema)",
		Tags:        []string{"shell", "desktop", "system"},
	}

	Minimal = Profile{
		Name:        "minimal",
		Description: "Minimo para devcontainer/CI (apenas shell)",
		Tags:        []string{"shell"},
		ExcludeTags: []string{"desktop", "system"},
	}

	Server = Profile{
		Name:        "server",
		Description: "Servidor sem desktop (shell + sistema)",
		Tags:        []string{"shell", "system"},
		ExcludeTags: []string{"desktop"},
	}

	WSL = Profile{
		Name:        "wsl",
		Description: "Ambiente WSL2 (shell + containers)",
		Tags:        []string{"shell", "wsl"},
		ExcludeTags: []string{"desktop", "system"},
	}
)

// All retorna todos os perfis disponiveis.
func All() []Profile {
	return []Profile{Full, Minimal, Server, WSL}
}

// ByName busca um perfil pelo nome.
// Retorna erro se o nome nao corresponder a nenhum perfil conhecido.
func ByName(name string) (Profile, error) {
	for _, p := range All() {
		if p.Name == name {
			return p, nil
		}
	}
	names := make([]string, len(All()))
	for i, p := range All() {
		names[i] = p.Name
	}
	return Profile{}, fmt.Errorf("perfil desconhecido: %q (disponiveis: %s)", name, strings.Join(names, ", "))
}
