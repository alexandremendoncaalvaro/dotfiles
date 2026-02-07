# dotfiles

Configurações para [Bluefin](https://projectbluefin.io) (Fedora Atomic). Roda uma vez e deixa tudo pronto.

## Instalar

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/ale/dotfiles/main/scripts/install.sh)"
```

Isso clona o repositório, instala Go via `brew` se necessário, compila e abre o TUI.

Já clonou? `make run` faz tudo.

## O que configura

| Módulo | O que faz |
|--------|-----------|
| **starship** | Instala o prompt [Starship](https://starship.rs), configura `.bashrc` e `.zshrc` |
| **cedilla-fix** | Corrige cedilha (`ç`) no Wayland/GNOME via `~/.XCompose` |
| **gnome-forge** | Auto-tiling com [Forge](https://github.com/forge-ext/forge). `Super+setas` = foco, `+Shift` = mover, `+Ctrl` = resize |
| **gnome-focus-mode** | `F11` = fullscreen + workspace exclusivo (estilo macOS) |
| **bluefin-update** | Atualiza rpm-ostree, Flatpak, firmware e Distrobox |

Na TUI você escolhe quais módulos quer — não precisa instalar tudo.

## Comandos

```bash
dotfiles apply            # Abre o TUI, escolha os módulos
dotfiles apply --headless # Aplica tudo sem interação
dotfiles status           # Mostra o que está instalado
dotfiles update           # Atualiza o dotfiles (git pull + rebuild)
```

O perfil é detectado automaticamente:

| Onde você está | Perfil | Módulos |
|----------------|--------|---------|
| Desktop Bluefin | `full` | Todos |
| Container / Distrobox | `minimal` | Só shell (starship) |
| Sem sessão gráfica | `server` | Shell + sistema |

Para forçar: `dotfiles apply -p minimal`

## Atualizar

```bash
dotfiles update
```

Ou manualmente:

```bash
cd ~/dotfiles && git pull && make build
```

## Contribuindo um módulo

1. Crie `internal/modules/nome/nome.go`
2. Implemente `Module`, `Checker` e `Applier` (e `Guard` se precisar pular em certos ambientes)
3. Registre em `cmd/dotfiles/main.go` com `reg.Register(nome.New())`
4. Adicione tag(s) (`shell`, `desktop`, `system`) para controle por perfil
5. Escreva testes usando `system.Mock` — veja qualquer módulo existente como exemplo

```bash
make test    # Roda os testes
make lint    # go vet
```
