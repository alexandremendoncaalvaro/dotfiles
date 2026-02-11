# blueprint

Configurações para [Bluefin](https://projectbluefin.io) (Fedora Atomic). Roda uma vez e deixa tudo pronto.

## Instalar

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/ale/blueprint/main/scripts/install.sh)"
```

Isso clona o repositório, instala Go via `brew` se necessário, compila e abre o TUI.

Já clonou? `make run` faz tudo.

## O que configura

| Módulo | O que faz |
|--------|-----------|
| **devcontainers** | Habilita dev mode, troca Docker CE por `podman-docker` (requer reboot) |
| **devbox** | Cria distrobox Ubuntu 24.04 com Node, Python, .NET, Java, Go, Rust via [mise](https://mise.run) |
| **starship** | Instala o prompt [Starship](https://starship.rs), configura `.bashrc` e `.zshrc` |
| **cedilla-fix** | Corrige cedilha (`ç`) no Wayland/GNOME via `~/.XCompose` |
| **tiling-shell** | Auto-tiling com [Tiling Shell](https://github.com/domferr/tilingshell) |
| **clipboard-indicator** | Histórico de clipboard no GNOME com [Clipboard Indicator](https://github.com/Tudmotu/gnome-shell-extension-clipboard-indicator) |
| **gnome-focus-mode** | `F11` = fullscreen + workspace exclusivo (estilo macOS) |
| **bluefin-update** | Atualiza rpm-ostree, Flatpak, firmware e Distrobox |

Na TUI você escolhe quais módulos quer — não precisa instalar tudo.

## Comandos

```bash
blueprint apply            # Abre o TUI, escolha os módulos
blueprint apply --headless # Aplica tudo sem interação
blueprint status           # Mostra o que está instalado
blueprint update           # Atualiza o blueprint (git pull + rebuild)
```

O perfil é detectado automaticamente:

| Onde você está | Perfil | Módulos |
|----------------|--------|---------|
| Desktop Bluefin | `full` | Todos |
| Container / Distrobox | `minimal` | Só shell (starship) |
| Sem sessão gráfica | `server` | Shell + sistema |

Para forçar: `blueprint apply -p minimal`

## VS Code + devbox

O módulo **devbox** cria o container e provisiona todas as ferramentas. Para conectar com o VS Code via "Attach to Running Container", rode **em cada máquina cliente** (Mac, Linux ou Windows):

```bash
curl -fsSL https://raw.githubusercontent.com/ale/blueprint/main/scripts/vscode-devbox.sh | bash
```

Isso cria um named container config que faz o VS Code conectar como `ale` (em vez de `root`). Só precisa rodar uma vez por máquina.

Depois: **VS Code > Remote Explorer > Dev Containers > Attach to Running Container > devbox**

> O blueprint mostra essa instrução automaticamente na tela de sumário ao aplicar o módulo devbox.

## Atualizar

```bash
blueprint update
```

Ou manualmente:

```bash
cd ~/blueprint && git pull && make build
```

## Contribuindo um módulo

1. Crie `internal/modules/nome/nome.go`
2. Implemente `Module`, `Checker` e `Applier` (e `Guard` se precisar pular em certos ambientes)
3. Registre em `cmd/blueprint/main.go` com `reg.Register(nome.New())`
4. Adicione tag(s) (`shell`, `desktop`, `system`) para controle por perfil
5. Escreva testes usando `system.Mock` — veja qualquer módulo existente como exemplo

```bash
make test    # Roda os testes
make lint    # go vet
```
