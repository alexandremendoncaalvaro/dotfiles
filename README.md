# dotfiles

CLI para configurar e manter um desktop [Bluefin](https://projectbluefin.io) (Fedora Atomic). Escrito em Go, com TUI interativo via [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## O que faz

O programa organiza configurações em **módulos** independentes, filtrados por **perfil**:

| Módulo | Tags | O que configura |
|--------|------|-----------------|
| `starship` | shell | Instala o prompt [Starship](https://starship.rs), cria symlink do config e adiciona init ao `.bashrc`/`.zshrc` |
| `cedilla-fix` | desktop | Corrige cedilha no Bluefin (Wayland/GNOME) via `~/.XCompose` |
| `gnome-forge` | desktop | Auto-tiling com [Forge](https://github.com/forge-ext/forge). Super+setas (foco), Super+Shift (mover), Super+Ctrl (resize) |
| `gnome-focus-mode` | desktop | F11 = fullscreen + workspace exclusivo (estilo macOS). Extensão GNOME Shell própria |
| `bluefin-update` | system | Atualiza rpm-ostree, Flatpak, firmware (fwupd) e Distrobox |

### Perfis

| Perfil | Tags incluídas | Excluídas | Caso de uso |
|--------|---------------|-----------|-------------|
| `full` | shell, desktop, system | — | Desktop Bluefin completo |
| `minimal` | shell | desktop, system | Devcontainer / CI |
| `server` | shell, system | desktop | Servidor sem desktop |

## Instalação

Uma linha — clona, instala Go se precisar, compila e abre o TUI:

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/ale/dotfiles/main/scripts/install.sh)"
```

Já clonou? Só rodar:

```bash
cd ~/dotfiles
make run
```

> No Bluefin o Go é instalado via `brew` automaticamente. Em distrobox, usa `/usr/local/go` se disponível.

## Uso

```bash
# Detecta o ambiente e abre o TUI com os módulos certos
dotfiles apply

# Forçar um perfil específico
dotfiles apply -p minimal

# Headless (sem TUI)
dotfiles apply --headless

# Simular sem executar
dotfiles apply --dry-run --headless

# Ver status dos módulos
dotfiles status

# Atualizar sistema (atalho para bluefin-update)
dotfiles update
```

O perfil é detectado automaticamente:

| Contexto | Perfil | Razão |
|----------|--------|-------|
| Container | `minimal` | Sem desktop, sem system |
| Sem sessão gráfica | `server` | Sem desktop |
| Desktop normal | `full` | Tudo disponível |

Use `--profile` / `-p` para sobrescrever.

## Estrutura

```
cmd/dotfiles/       → Entry point
internal/
  module/           → Interfaces de domínio (Module, System, Guard, Checker, Applier)
  modules/          → Implementações dos módulos
  profile/          → Perfis e resolução de tags
  orchestrator/     → Guard → Check → Apply pipeline
  system/           → Implementações de System (Real, Mock, DryRun)
  cli/              → Comandos Cobra
  tui/              → Interface Bubble Tea
configs/            → Arquivos de configuração (starship.toml)
```

## Adicionando um módulo

1. Crie `internal/modules/nome/nome.go` implementando `module.Module` + as interfaces necessárias (`Checker`, `Applier`, opcionalmente `Guard`)
2. Registre em `cmd/dotfiles/main.go` com `reg.Register(nome.New())`

O módulo só precisa implementar o que usa — o orchestrator detecta as interfaces por type assertion.

## Testes

```bash
make test
```
