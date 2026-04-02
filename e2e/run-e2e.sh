#!/usr/bin/env bash
# =============================================================================
# Full E2E test for ssh-wrapper using docker compose.
# Generates a fresh SSH key pair on every run — no secrets to manage.
#
# Usage:
#   ./e2e/run_e2e.sh
#
# Works locally and in GitHub Actions (no stored secrets needed).
# =============================================================================
set -euo pipefail

# ---------- colors ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
pass() { echo -e "${GREEN}[PASS]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; FAILURES=$((FAILURES + 1)); }
info() { echo -e "${YELLOW}[INFO]${NC} $*"; }
FAILURES=0

# ---------- cleanup on exit ----------
cleanup() {
  info "Stopping containers..."
  docker compose -f test-compose.yaml down -v --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

# =============================================================================
# STEP 0: Check for required tools
# =============================================================================
if ! command -v docker &>/dev/null; then
  fail "docker is required but not installed"
  exit 1
fi

rm -rf test-data/keys test-data/repos test-data/logs

# =============================================================================
# STEP 1: Generate fresh SSH key pair
# =============================================================================
info "Generating SSH key pair..."
mkdir -p test-data/keys test-data/repos test-data/logs

# Remove any leftover key from previous run
rm -f test-data/keys/id_ed25519 test-data/keys/id_ed25519.pub

ssh-keygen -t ed25519 -f test-data/keys/id_ed25519 -N "" -C "e2e-test" -q

# =============================================================================
# STEP 2: Create a bare test repo in the git-server repos volume
# =============================================================================
info "Initialising bare test repo..."
mkdir -p test-data/repos/repo1
git init --bare test-data/repos/repo1 --quiet

# =============================================================================
# STEP 3: Bring up docker compose
#   - git-server starts and reads the public key from test-data/keys/
#   - test-app waits for git-server healthcheck before becoming ready
# =============================================================================
info "Starting docker compose (building if needed)..."
docker compose -f test-compose.yaml up -d --build

info "Waiting for services to be healthy..."
sleep 3

# =============================================================================
# STEP 4: Run tests inside test-app via docker compose exec -T
# =============================================================================
EXEC="docker compose -f test-compose.yaml exec -T --user=1000 test-app"

# ---- Test 1: git clone from allowed host ----
info "Test 1: git clone from allowed host (git-server)"
if $EXEC /bin/sh -c '
  set -e
  rm -rf /tmp/repo1
  git clone git@git-server:/git-server/repos/repo1 /tmp/repo1
'; then
  pass "Test 1: git clone from allowed host succeeded"
else
  fail "Test 1: git clone from allowed host FAILED"
fi

# ---- Test 2: git commit + push ----
info "Test 2: git commit + push to allowed host"
if $EXEC /bin/sh -c '
  set -e
  cd /tmp/repo1
  echo "e2e marker $(date)" > marker.txt
  git add marker.txt
  git commit -m "e2e: marker commit"
  git push origin HEAD
'; then
  pass "Test 2: git push to allowed host succeeded"
else
  fail "Test 2: git push to allowed host FAILED"
fi

# ---- Test 3: git pull ----
info "Test 3: git pull from allowed host"
if $EXEC /bin/sh -c '
  set -e
  cd /tmp/repo1
  git pull
'; then
  pass "Test 3: git pull succeeded"
else
  fail "Test 3: git pull FAILED"
fi

# ---- Test 4: SSH to unknown host is blocked ----
info "Test 4: SSH to disallowed host should be blocked by ssh-wrapper"
if $EXEC /bin/sh -c '
  output=$(ssh git@not-allowed-host echo hi 2>&1 || true)
  echo "$output"
  echo "$output" | grep -qi "Access Denied"
'; then
  pass "Test 4: Disallowed host correctly blocked by ssh-wrapper"
else
  fail "Test 4: Disallowed host was NOT blocked"
fi

# =============================================================================
# STEP 5: Assert on the log file written to the host volume
# =============================================================================
info "Checking log file..."
LOG_FILE="test-data/logs/ssh-wrapper.log"

if [[ -s "$LOG_FILE" ]]; then
  pass "Log file exists and is non-empty"
else
  fail "Log file is missing or empty: $LOG_FILE"
fi

info "Log contents:"
cat "$LOG_FILE"
echo ""

if grep -q "git-server" "$LOG_FILE"; then
  pass "Log contains git-server SSH activity"
else
  fail "Log does NOT contain expected git-server activity"
fi

# =============================================================================
# Summary
# =============================================================================
echo ""
if [[ $FAILURES -eq 0 ]]; then
  echo -e "${GREEN}━━━ All tests passed ━━━${NC}"
  exit 0
else
  echo -e "${RED}━━━ $FAILURES test(s) failed ━━━${NC}"
  exit 1
fi
