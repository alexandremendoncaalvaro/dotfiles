#!/usr/bin/env bash
set -euo pipefail

# Bootstrap idempotente do blueprint.
# Cada etapa verifica se já está resolvida antes de agir.
# Se um pré-requisito faltar, cobra antes de prosseguir.
#
# Uso remoto:
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/ale/blueprint/main/scripts/install.sh)"
#
# Uso local:
#   ./scripts/install.sh

REPO="https://github.com/ale/blueprint.git"
# Se estamos rodando de dentro de um clone, usa o dir atual. Senao, usa ~/blueprint.
CURRENT_GIT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo "")
if [[ -n "$CURRENT_GIT_ROOT" ]]; then
    DEFAULT_INSTALL_DIR="$CURRENT_GIT_ROOT"
else
    DEFAULT_INSTALL_DIR="$HOME/blueprint"
fi

INSTALL_DIR="${BLUEPRINT_DIR:-$DEFAULT_INSTALL_DIR}"
LOCAL_BIN="$HOME/.local/bin"

info() { printf '\033[1;34m  ·\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  ✓\033[0m %s\n' "$*"; }
skip() { printf '\033[0;90m  – %s (já ok)\033[0m\n' "$*"; }
fail() { printf '\033[1;31m  ✗ %s\033[0m\n' "$*" >&2; exit 1; }

# ── 0. Pré-requisitos obrigatórios ──────────────────────────────
check_prereqs() {
    local missing=()
    command -v git  &>/dev/null || missing+=(git)
    command -v curl &>/dev/null || missing+=(curl)
    command -v make &>/dev/null || missing+=(make)

    if [[ ${#missing[@]} -gt 0 ]]; then
        fail "Faltam dependências: ${missing[*]}. Instale e rode novamente."
    fi
}

# ── 1. Go ───────────────────────────────────────────────────────
ensure_go() {
    # Já no PATH?
    if command -v go &>/dev/null; then
        skip "Go $(go version | awk '{print $3}')"
        return
    fi

    # Instalado mas fora do PATH? (comum em distrobox)
    if [[ -x /usr/local/go/bin/go ]]; then
        export PATH="/usr/local/go/bin:$PATH"
        skip "Go $(go version | awk '{print $3}') (/usr/local/go)"
        return
    fi

    # Homebrew/Linuxbrew disponível? (jeito Bluefin)
    if command -v brew &>/dev/null; then
        info "Instalando Go via brew..."
        brew install go
        ok "Go instalado via brew"
        return
    fi

    fail "Go não encontrado. Instale com: brew install go"
}

# ── 2. Repositório ─────────────────────────────────────────────
ensure_repo() {
    if [[ -d "$INSTALL_DIR/.git" ]]; then
        local LOCAL_HEAD REMOTE_HEAD
        LOCAL_HEAD=$(git -C "$INSTALL_DIR" rev-parse HEAD 2>/dev/null)
        git -C "$INSTALL_DIR" fetch --quiet 2>/dev/null || true
        REMOTE_HEAD=$(git -C "$INSTALL_DIR" rev-parse FETCH_HEAD 2>/dev/null || echo "$LOCAL_HEAD")

        if [[ "$LOCAL_HEAD" == "$REMOTE_HEAD" ]]; then
            skip "Repo $INSTALL_DIR"
        else
            info "Atualizando $INSTALL_DIR..."
            git -C "$INSTALL_DIR" pull --ff-only
            ok "Repo atualizado"
        fi
    else
        info "Clonando em $INSTALL_DIR..."
        git clone "$REPO" "$INSTALL_DIR"
        ok "Repo clonado"
    fi
}

# ── 3. Build ───────────────────────────────────────────────────
ensure_build() {
    command -v go &>/dev/null || fail "Go não encontrado. A etapa 1 falhou?"

    local BIN="$INSTALL_DIR/bin/blueprint"

    # Binário existe e é mais novo que qualquer .go?
    if [[ -x "$BIN" ]]; then
        local NEWEST_SRC
        NEWEST_SRC=$(find "$INSTALL_DIR" -name '*.go' -newer "$BIN" 2>/dev/null | head -1)
        if [[ -z "$NEWEST_SRC" ]]; then
            skip "Binário atualizado"
            return
        fi
        info "Fonte modificado, recompilando..."
    else
        info "Compilando..."
    fi

    cd "$INSTALL_DIR"
    make build
    ok "Binário: $BIN"
}

# ── 4. Link no PATH ───────────────────────────────────────────
ensure_link() {
    [[ -x "$INSTALL_DIR/bin/blueprint" ]] || fail "Binário não encontrado. A etapa 3 falhou?"

    mkdir -p "$LOCAL_BIN"
    local LINK="$LOCAL_BIN/blueprint"

    if [[ -L "$LINK" ]] && [[ "$(readlink -f "$LINK")" == "$(readlink -f "$INSTALL_DIR/bin/blueprint")" ]]; then
        skip "Link ~/.local/bin/blueprint"
    else
        ln -sf "$INSTALL_DIR/bin/blueprint" "$LINK"
        ok "Link criado: $LINK"
    fi

    if [[ ":$PATH:" != *":$LOCAL_BIN:"* ]]; then
        export PATH="$LOCAL_BIN:$PATH"
    fi
}

# ── Main ───────────────────────────────────────────────────────
main() {
    echo
    printf '\033[1m  Blueprint Setup\033[0m\n'
    echo

    check_prereqs
    ensure_go
    ensure_repo
    ensure_build
    ensure_link

    echo
    ok "Tudo pronto. Abrindo TUI..."
    echo

    # Garante que usamos o binario que acabamos de compilar
    export BLUEPRINT_DIR="$INSTALL_DIR"
    exec "$INSTALL_DIR/bin/blueprint" apply
}

main "$@"
