#!/usr/bin/env bash
# SDD Workflow Installer
# Usage: curl -sSL https://raw.githubusercontent.com/rechedev9/sdd-workflow/master/install.sh | bash

set -euo pipefail

REPO_URL="https://github.com/rechedev9/sdd-workflow.git"
CLAUDE_DIR="${HOME}/.claude"
BACKUP_DIR="${CLAUDE_DIR}/.sdd-backup-$(date +%Y%m%d-%H%M%S)"
TMP_DIR=""

# ── Colours ────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}  →${RESET} $*"; }
success() { echo -e "${GREEN}  ✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}  !${RESET} $*"; }
fatal()   { echo -e "${RED}  ✗ ERROR:${RESET} $*" >&2; exit 1; }
header()  { echo -e "\n${BOLD}$*${RESET}"; }

cleanup() {
  if [[ -n "${TMP_DIR}" && -d "${TMP_DIR}" ]]; then
    rm -rf "${TMP_DIR}"
  fi
}
trap cleanup EXIT

# ── Banner ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}  SDD Workflow Installer${RESET}"
echo -e "  Spec-Driven Development for Claude Code"
echo -e "  https://github.com/rechedev9/sdd-workflow"
echo ""

# ── Prerequisites ──────────────────────────────────────────────────────────
header "Checking prerequisites..."

command -v git >/dev/null 2>&1 || fatal "git is required. Install it from https://git-scm.com"
success "git found: $(git --version)"

if ! command -v claude >/dev/null 2>&1; then
  warn "Claude Code CLI not found in PATH."
  warn "Install it from: https://claude.ai/claude-code"
  warn "The skills will still be installed — just make sure Claude Code is set up before using SDD."
else
  success "Claude Code found: $(claude --version 2>/dev/null || echo 'installed')"
fi

# ── Get source files ───────────────────────────────────────────────────────
header "Getting SDD Workflow files..."

# Detect if we're already running from inside the repo
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" 2>/dev/null && pwd || pwd)"

if [[ -d "${SCRIPT_DIR}/skills" && -d "${SCRIPT_DIR}/commands" ]]; then
  info "Running from local repo: ${SCRIPT_DIR}"
  SRC_DIR="${SCRIPT_DIR}"
else
  info "Cloning repository..."
  TMP_DIR="$(mktemp -d)"
  git clone --depth 1 --quiet "${REPO_URL}" "${TMP_DIR}/sdd-workflow"
  SRC_DIR="${TMP_DIR}/sdd-workflow"
  success "Repository cloned"
fi

# ── Back up existing files ─────────────────────────────────────────────────
header "Backing up existing ~/.claude files..."

BACKED_UP=0
for dir in skills/sdd skills/frameworks skills/analysis skills/knowledge skills/workflows commands; do
  TARGET="${CLAUDE_DIR}/${dir}"
  if [[ -d "${TARGET}" ]]; then
    mkdir -p "${BACKUP_DIR}/${dir}"
    cp -r "${TARGET}/." "${BACKUP_DIR}/${dir}/"
    BACKED_UP=1
  fi
done

if [[ "${BACKED_UP}" -eq 1 ]]; then
  success "Existing files backed up to: ${BACKUP_DIR}"
else
  info "No existing SDD files found — clean install"
fi

# ── Install skills ─────────────────────────────────────────────────────────
header "Installing skills..."

install_skill_group() {
  local GROUP="$1"
  local COUNT=0
  local SRC="${SRC_DIR}/skills/${GROUP}"
  local DST="${CLAUDE_DIR}/skills/${GROUP}"

  if [[ ! -d "${SRC}" ]]; then
    return
  fi

  mkdir -p "${DST}"
  for skill_dir in "${SRC}"/*/; do
    skill_name="$(basename "${skill_dir}")"
    mkdir -p "${DST}/${skill_name}"
    cp -r "${skill_dir}/." "${DST}/${skill_name}/"
    COUNT=$((COUNT + 1))
  done

  success "${COUNT} ${GROUP} skills installed → ${DST}"
}

install_skill_group "sdd"
install_skill_group "frameworks"
install_skill_group "analysis"
install_skill_group "knowledge"
install_skill_group "workflows"

# Ensure learned/ directory exists (skills accumulate here at runtime)
mkdir -p "${CLAUDE_DIR}/skills/learned"
success "learned/ directory ready → ${CLAUDE_DIR}/skills/learned"

# ── Install commands ───────────────────────────────────────────────────────
header "Installing commands..."

COMMANDS_DST="${CLAUDE_DIR}/commands"
mkdir -p "${COMMANDS_DST}"
CMD_COUNT=0
for cmd_file in "${SRC_DIR}/commands/"*.md; do
  cp "${cmd_file}" "${COMMANDS_DST}/"
  CMD_COUNT=$((CMD_COUNT + 1))
done
success "${CMD_COUNT} commands installed → ${COMMANDS_DST}"

# ── CLAUDE.md reminder ─────────────────────────────────────────────────────
header "CLAUDE.md setup..."

# CLAUDE.md lives at the project level, not globally.
# Check for a global one as a bonus, but don't warn if missing.
CLAUDE_MD="${CLAUDE_DIR}/CLAUDE.md"
if [[ -f "${CLAUDE_MD}" ]]; then
  success "Global CLAUDE.md found at ${CLAUDE_MD}"
else
  info "CLAUDE.md is project-level — add one to each repo you use SDD in."
  info "See: https://github.com/rechedev9/sdd-workflow/blob/master/docs/07-configuration.md"
fi

# ── Summary ────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}  Installation complete!${RESET}"
echo ""
echo -e "  ${GREEN}SDD skills installed:${RESET}"
echo "    • 11 SDD phase skills  (init → archive)"
echo "    • 14 framework skills  (React 19, Tailwind 4, TypeScript, …)"
echo "    •  8 support skills    (analysis, knowledge, workflows)"
echo "    • 17 slash commands"
echo ""
echo -e "  ${GREEN}Quick start:${RESET}"
echo ""
echo "    cd /path/to/your/project"
echo "    claude"
echo "    /sdd:init"
echo "    /sdd:new my-feature \"What I want to build\""
echo ""
echo -e "  ${GREEN}Docs:${RESET} https://github.com/rechedev9/sdd-workflow"
echo ""
