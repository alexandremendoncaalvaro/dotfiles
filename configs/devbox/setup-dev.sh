#!/usr/bin/env bash
# setup-dev.sh — Provisiona o devbox com todas as toolchains via mise
#
# Uso:
#   distrobox enter devbox -- bash <blueprint-repo>/configs/devbox/setup-dev.sh
#
# Idempotente: pode rodar várias vezes sem efeito colateral.
set -euo pipefail

# ── Fix SSL certs (host Fedora paths don't exist in Ubuntu) ──────────
export SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
export SSL_CERT_DIR=/etc/ssl/certs

# ── Cores ────────────────────────────────────────────────────────────
BOLD='\033[1m'
GREEN='\033[1;32m'
BLUE='\033[1;34m'
RED='\033[1;31m'
RESET='\033[0m'

step()  { echo -e "\n${BLUE}${BOLD}▸ $*${RESET}"; }
ok()    { echo -e "${GREEN}✓ $*${RESET}"; }
fail()  { echo -e "${RED}✗ $*${RESET}"; }

# ═════════════════════════════════════════════════════════════════════
# 1. System packages
# ═════════════════════════════════════════════════════════════════════
step "System packages"
sudo apt-get update -qq
sudo apt-get install -y -qq \
  build-essential gcc g++ clang make cmake \
  git curl wget unzip zip jq file \
  zsh \
  libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev \
  libncursesw5-dev libxml2-dev libxmlsec1-dev libffi-dev liblzma-dev \
  tk-dev xz-utils ca-certificates \
  2>&1 | tail -1
ok "System packages installed"

# ═════════════════════════════════════════════════════════════════════
# 2. Starship prompt
# ═════════════════════════════════════════════════════════════════════
step "Starship"
if ! command -v starship &>/dev/null; then
  curl -sS https://starship.rs/install.sh | sh -s -- -y >/dev/null 2>&1
fi
ok "Starship $(starship --version 2>/dev/null | awk '{print $2}')"

# ═════════════════════════════════════════════════════════════════════
# 3. mise
# ═════════════════════════════════════════════════════════════════════
step "mise"
if ! command -v mise &>/dev/null; then
  curl https://mise.run | sh
fi
eval "$(~/.local/bin/mise activate bash)"
ok "mise $(mise --version 2>/dev/null)"

# ═════════════════════════════════════════════════════════════════════
# 4. Zsh + Starship + mise integration
# ═════════════════════════════════════════════════════════════════════
step "Shell config"
cat > "$HOME/.zshrc" << 'ZSHRC'
# ── mise ─────────────────────────────────────────────────────────
eval "$(~/.local/bin/mise activate zsh)"

# ── starship ─────────────────────────────────────────────────────
eval "$(starship init zsh)"

# ── aliases ──────────────────────────────────────────────────────
alias ll='ls -lah --color=auto'
alias la='ls -A --color=auto'
ZSHRC

# bashrc fallback
grep -q 'mise activate' "$HOME/.bashrc" 2>/dev/null || \
  echo 'eval "$(~/.local/bin/mise activate bash)"' >> "$HOME/.bashrc"

sudo chsh -s "$(which zsh)" "$(whoami)" 2>/dev/null || true
ok "Zsh configured"

# ═════════════════════════════════════════════════════════════════════
# 5. Install toolchains
# ═════════════════════════════════════════════════════════════════════

# ── Web ──────────────────────────────────────────────────────────
step "Web stack"
mise use -g node@lts
mise use -g bun@latest
mise use -g deno@latest
ok "Web stack"

# ── Python ───────────────────────────────────────────────────────
step "Python stack"
mise use -g python@latest
mise use -g uv@latest
ok "Python stack"

# ── .NET ─────────────────────────────────────────────────────────
step ".NET stack"
mise use -g dotnet@latest
ok ".NET stack"

# ── Java ─────────────────────────────────────────────────────────
step "Java stack"
mise use -g java@temurin-21
mise use -g maven@latest
mise use -g gradle@latest
ok "Java stack"

# ── Low-level ────────────────────────────────────────────────────
step "Low-level stack"
mise use -g go@latest
mise use -g rust@latest
# gcc + clang → already installed via apt
ok "Low-level stack"

# ═════════════════════════════════════════════════════════════════════
# Done
# ═════════════════════════════════════════════════════════════════════
echo ""
echo -e "${BOLD}══════════════════════════════════════════════════════${RESET}"
echo -e "${GREEN}${BOLD}  devbox ready!${RESET}"
echo -e "${BOLD}══════════════════════════════════════════════════════${RESET}"
echo ""
echo "Toolchains instaladas:"
mise ls --current 2>/dev/null
echo ""
echo "Uso:"
echo "  distrobox enter devbox"
echo "  Per-project: mise use node@20 (gera mise.toml ou .tool-versions)"
echo "  VS Code: Attach to Running Container → devbox"
