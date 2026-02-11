#!/usr/bin/env bash
# vscode-devbox.sh — Configura o VS Code local para conectar no devbox como ale.
#
# O distrobox cria containers com --user root:root (necessario pro entrypoint).
# O VS Code "Attach to Running Container" usa esse user por padrao.
# Este script cria o named container config que faz o override para "ale".
#
# Uso (em qualquer maquina onde roda o VS Code):
#   curl -fsSL https://raw.githubusercontent.com/ale/blueprint/main/scripts/vscode-devbox.sh | bash
#
# Idempotente: pode rodar varias vezes.
set -euo pipefail

CONFIG='{"remoteUser":"ale"}'
FILE="devbox.json"

case "$(uname -s)" in
  Darwin)
    DIR="$HOME/Library/Application Support/Code/User/globalStorage/ms-vscode-remote.remote-containers/nameConfigs"
    ;;
  Linux)
    DIR="$HOME/.config/Code/User/globalStorage/ms-vscode-remote.remote-containers/nameConfigs"
    ;;
  MINGW*|MSYS*|CYGWIN*)
    DIR="$APPDATA/Code/User/globalStorage/ms-vscode-remote.remote-containers/nameConfigs"
    ;;
  *)
    echo "OS nao suportado: $(uname -s)" >&2
    exit 1
    ;;
esac

mkdir -p "$DIR"
echo "$CONFIG" > "$DIR/$FILE"
echo "VS Code devbox config criado em: $DIR/$FILE"
