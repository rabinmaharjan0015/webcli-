#!/usr/bin/env bash
set -euo pipefail

APP="webcli"
CONFIG_DIR="${HOME}/.webcli"
BINARY_DIRS=("/usr/local/bin" "${HOME}/.local/bin" "${HOME}/go/bin" "${HOME}/bin")

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${BLUE}info${NC} $1"; }
ok()    { echo -e "${GREEN}ok${NC}   $1"; }
warn()  { echo -e "${YELLOW}warn${NC} $1"; }
error() { echo -e "${RED}error${NC} $1"; }

echo ""
echo "╭────────────────────────────────────────────╮"
echo "│       WebCLI Uninstaller                   │"
echo "╰────────────────────────────────────────────╯"
echo ""

# Find and remove binary
FOUND=""
for dir in "${BINARY_DIRS[@]}"; do
  if [ -f "${dir}/${APP}" ]; then
    rm -f "${dir}/${APP}" 2>/dev/null && ok "Removed ${dir}/${APP}" || error "Could not remove ${dir}/${APP}"
    FOUND="yes"
  fi
done

if command -v "${APP}" &>/dev/null; then
  BIN_PATH=$(command -v "${APP}")
  rm -f "${BIN_PATH}" 2>/dev/null && ok "Removed ${BIN_PATH}" || error "Could not remove ${BIN_PATH}"
  FOUND="yes"
fi

if [ -z "${FOUND}" ]; then
  warn "No ${APP} binary found in PATH."
  echo "  Checked: ${BINARY_DIRS[*]}"
fi

# Remove npm wrapper (if installed globally)
NPM_BIN=$(npm bin -g 2>/dev/null || true)
if [ -n "${NPM_BIN}" ] && [ -f "${NPM_BIN}/${APP}" ]; then
  rm -f "${NPM_BIN}/${APP}" 2>/dev/null && ok "Removed npm global binary" || warn "Could not remove npm global binary"
fi

# Remove config and data
echo ""
if [ -d "${CONFIG_DIR}" ]; then
  echo "Remove all config, memory, and data files?"
  echo "  Location: ${CONFIG_DIR}"
  echo -n "  [y/N] "
  read -r CONFIRM
  if [ "${CONFIRM}" = "y" ] || [ "${CONFIRM}" = "Y" ]; then
    rm -rf "${CONFIG_DIR}" && ok "Removed ${CONFIG_DIR}" || error "Could not remove ${CONFIG_DIR}"
  else
    warn "Skipped: ${CONFIG_DIR}"
    echo "  Remove manually: rm -rf ${CONFIG_DIR}"
  fi
else
  info "No config directory found at ${CONFIG_DIR}"
fi

echo ""
if command -v "${APP}" &>/dev/null; then
  warn "${APP} is still available in PATH. Try running: which ${APP}"
  echo "  You may need to restart your terminal or remove it manually."
  echo "Done (partial)"
else
  ok "${APP} has been uninstalled."
fi
echo ""
