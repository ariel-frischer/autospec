#!/usr/bin/env bash
#
# Autospec Quickstart Demo
# ========================
# Run this script to see autospec's core workflow in action.
# Works on any git repository.
#
# Usage:
#   ./quickstart-demo.sh              # Interactive mode (recommended)
#   ./quickstart-demo.sh --dry-run    # Show commands without executing
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

print_header() {
    echo -e "\n${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${CYAN}  $1${NC}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_step() {
    echo -e "${GREEN}▶${NC} ${BOLD}$1${NC}"
}

print_cmd() {
    echo -e "  ${YELLOW}\$${NC} $1"
}

run_cmd() {
    print_cmd "$1"
    if [[ "$DRY_RUN" == "false" ]]; then
        eval "$1"
    fi
}

pause() {
    if [[ "$DRY_RUN" == "false" ]]; then
        echo -e "\n${CYAN}Press Enter to continue...${NC}"
        read -r
    fi
}

# ============================================================================
# DEMO START
# ============================================================================

cat << 'EOF'

    ▄▀█ █ █ ▀█▀ █▀█ █▀ █▀█ █▀▀ █▀▀
    █▀█ █▄█  █  █▄█ ▄█ █▀▀ ██▄ █▄▄

    Spec-Driven Development Automation
    ===================================

    This demo shows the core autospec workflow:

    1. Check dependencies (doctor)
    2. Initialize project (init)
    3. Create project principles (constitution)
    4. Generate spec → plan → tasks (prep)
    5. View status (st)
    6. Execute implementation (implement)

EOF

pause

# ----------------------------------------------------------------------------
print_header "1. CHECK DEPENDENCIES"
# ----------------------------------------------------------------------------

print_step "Verify all prerequisites are installed"
run_cmd "autospec doctor"

pause

# ----------------------------------------------------------------------------
print_header "2. INITIALIZE PROJECT"
# ----------------------------------------------------------------------------

print_step "Set up autospec in your repository"
echo -e "  Creates: ${CYAN}.autospec/${NC} directory with config and commands\n"
run_cmd "autospec init"

pause

# ----------------------------------------------------------------------------
print_header "3. PROJECT CONSTITUTION (Optional)"
# ----------------------------------------------------------------------------

print_step "Define project-wide principles that guide all specifications"
echo -e "  Creates: ${CYAN}.autospec/memory/constitution.yaml${NC}"
echo -e "  ${YELLOW}Note:${NC} This launches a Claude session for interactive Q&A\n"
print_cmd "autospec constitution"
echo -e "  ${CYAN}(Skipped in demo - run manually when ready)${NC}"

pause

# ----------------------------------------------------------------------------
print_header "4. CORE WORKFLOW OPTIONS"
# ----------------------------------------------------------------------------

cat << 'EOF'
There are several ways to run the workflow:

┌─────────────────────────────────────────────────────────────────────────────┐
│  COMMAND                              │  STAGES                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  autospec run -a "description"        │  specify → plan → tasks → implement │
│  autospec all "description"           │  (shortcut for -a)                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  autospec prep "description"          │  specify → plan → tasks (no impl)   │
│  autospec run -spt "description"      │  (same as prep)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  autospec run -s "description"        │  specify only                       │
│  autospec specify "description"       │  (same as -s)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  autospec implement                   │  implementation only                │
│  autospec implement --tasks           │  (per-task isolation)               │
└─────────────────────────────────────────────────────────────────────────────┘

EOF

pause

# ----------------------------------------------------------------------------
print_header "5. RECOMMENDED WORKFLOW: ITERATIVE APPROACH"
# ----------------------------------------------------------------------------

print_step "Step A: Generate specification only"
print_cmd 'autospec run -s "Add user authentication with OAuth"'
echo -e "  ${CYAN}→ Creates: specs/001-user-auth/spec.yaml${NC}"
echo -e "  ${CYAN}→ Creates branch: spec/001-user-auth${NC}\n"

print_step "Step B: Review and edit spec.yaml as needed"
echo -e "  ${CYAN}→ Refine requirements, add edge cases, clarify scope${NC}\n"

print_step "Step C: Continue with remaining stages"
print_cmd "autospec run -pti"
echo -e "  ${CYAN}→ Creates: plan.yaml, tasks.yaml${NC}"
echo -e "  ${CYAN}→ Executes implementation${NC}\n"

pause

# ----------------------------------------------------------------------------
print_header "6. EXAMPLE: FULL WORKFLOW DEMO"
# ----------------------------------------------------------------------------

echo -e "${YELLOW}The following commands would run a complete workflow:${NC}\n"

print_step "Option A: All stages at once (fast)"
print_cmd 'autospec run -a -y "Add a health check endpoint at /health"'
echo ""

print_step "Option B: Planning only, then implement separately (recommended)"
print_cmd 'autospec prep "Add rate limiting to API endpoints"'
print_cmd "# Review specs/00X-rate-limiting/*.yaml"
print_cmd "autospec implement"
echo ""

print_step "Option C: Stage by stage (maximum control)"
print_cmd 'autospec specify "Add caching layer for database queries"'
print_cmd "autospec plan"
print_cmd "autospec tasks"
print_cmd "autospec implement --tasks  # Per-task isolation"
echo ""

pause

# ----------------------------------------------------------------------------
print_header "7. MONITORING PROGRESS"
# ----------------------------------------------------------------------------

print_step "Check current status and task progress"
run_cmd "autospec st"

echo ""
print_step "Verbose status with all details"
print_cmd "autospec st -v"

pause

# ----------------------------------------------------------------------------
print_header "8. ADVANCED: IMPLEMENTATION MODES"
# ----------------------------------------------------------------------------

cat << 'EOF'
Control context isolation during implementation:

┌────────────────────────────────────────────────────────────────────────────┐
│  MODE           │  FLAG              │  USE CASE                          │
├────────────────────────────────────────────────────────────────────────────┤
│  Phase (default)│  --phases          │  Balanced cost/context             │
│  Task           │  --tasks           │  Complex tasks, max isolation      │
│  Single         │  --single-session  │  Small specs, simple tasks         │
└────────────────────────────────────────────────────────────────────────────┘

Resume options:
  autospec implement --from-phase 3     # Resume from phase 3
  autospec implement --from-task T005   # Resume from task T005
  autospec implement --task T003        # Run only task T003

EOF

pause

# ----------------------------------------------------------------------------
print_header "9. OPTIONAL STAGES"
# ----------------------------------------------------------------------------

cat << 'EOF'
Additional stages for refinement:

  autospec clarify      # Refine spec with Q&A (-r flag)
  autospec checklist    # Generate validation checklist (-l flag)
  autospec analyze      # Cross-artifact consistency check (-z flag)

Example: Full workflow with all optional stages:
  autospec run -a -r -l -z "Add payment processing"
  # Or: autospec run -arlz "Add payment processing"

EOF

pause

# ----------------------------------------------------------------------------
print_header "QUICK REFERENCE"
# ----------------------------------------------------------------------------

cat << 'EOF'
ESSENTIAL COMMANDS:
  autospec doctor                    # Check dependencies
  autospec init                      # Initialize project
  autospec constitution              # Create project principles
  autospec run -a "description"      # Full workflow
  autospec prep "description"        # Plan only (no implementation)
  autospec implement                 # Execute implementation
  autospec st                        # Show status

STAGE FLAGS (for 'run' command):
  -s  specify      -p  plan        -t  tasks       -i  implement
  -a  all (-spti)  -n  constitution -r  clarify    -l  checklist
  -z  analyze      -y  skip confirmations

IMPLEMENTATION MODES:
  --phases         One session per phase (default)
  --tasks          One session per task (max isolation)
  --single-session One session for everything

MORE INFO:
  autospec --help           # Full command reference
  autospec <command> --help # Command-specific help

EOF

echo -e "${GREEN}${BOLD}Demo complete!${NC} Run these commands in your repository to get started.\n"
