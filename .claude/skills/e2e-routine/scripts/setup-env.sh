#!/bin/bash
# Cloud environment setup script for the Bytebase e2e routine.
#
# Paste this file into the "Setup script" field of the cloud environment, or
# have the field run it from the repo:
#   curl -fsSL https://raw.githubusercontent.com/bytebase/bytebase/main/.claude/skills/e2e-routine/scripts/setup-env.sh | bash
#
# It runs as root on Ubuntu 24.04 before Claude Code starts. It must exit 0 and
# finish in under about five minutes, or the environment is not cached and the
# whole script re-runs on every session. It installs only toolchain; the repo
# build happens per run (build.sh) because `go mod download` plus `pnpm install`
# would not fit the cache window.
#
# Network: the environment needs Custom network access = default allowlist plus
#   cdn.playwright.dev
#   playwright.download.prss.microsoft.com
# for the Chromium download. nodejs.org, proxy.golang.org, registry.npmjs.org
# and archive.ubuntu.com are already in the default allowlist.

set -u
export DEBIAN_FRONTEND=noninteractive

NODE_VERSION=24.18.1          # .github/workflows/ci.yml
PNPM_VERSION=11.17.0          # frontend/package.json packageManager
GO_VERSION=1.27.1             # go.mod
PLAYWRIGHT_VERSION=1.59.1     # frontend/package.json @playwright/test; the harness re-syncs the exact build on first run

step() { printf '\n==> %s\n' "$*"; }

step "System packages: Postgres server binaries (Bytebase runs pg_ctl/initdb from PATH), psql, UTF-8 locale"
apt-get update -qq || true
apt-get install -y -qq postgresql-16 postgresql-client-16 locales procps lsof xz-utils >/dev/null || true
locale-gen en_US.UTF-8 >/dev/null 2>&1 || true
ls -d /usr/lib/postgresql/*/bin 2>/dev/null || echo "WARNING: no /usr/lib/postgresql/*/bin; the harness cannot boot Bytebase"

step "bytebase system user (Bytebase refuses to run as root without it; sessions run as root)"
if ! id bytebase >/dev/null 2>&1; then
  getent group bytebase >/dev/null || addgroup --gid 113 --system bytebase || addgroup --system bytebase || true
  adduser --uid 113 --system --ingroup bytebase --no-create-home bytebase \
    || adduser --system --ingroup bytebase --no-create-home bytebase || true
fi
id bytebase || echo "WARNING: no bytebase user; Bytebase will refuse to start as root"

step "Node ${NODE_VERSION} under /opt/node24 (the image ships 20-22)"
if [ ! -x /opt/node24/bin/node ]; then
  curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" -o /tmp/node24.tar.xz \
    && mkdir -p /opt/node24 \
    && tar -xJf /tmp/node24.tar.xz -C /opt/node24 --strip-components=1 \
    && rm -f /tmp/node24.tar.xz || true
fi
export PATH=/opt/node24/bin:$PATH
/opt/node24/bin/npm i -g "pnpm@${PNPM_VERSION}" >/dev/null 2>&1 || true

step "Go ${GO_VERSION} toolchain (downloaded through proxy.golang.org into the module cache)"
( GOTOOLCHAIN="go${GO_VERSION}" go version || true ) &
GO_PID=$!

step "Playwright Chromium ${PLAYWRIGHT_VERSION} + OS libraries"
npx --yes "playwright@${PLAYWRIGHT_VERSION}" install --with-deps chromium || true

wait $GO_PID || true

step "Login-shell defaults (scripts also set these themselves)"
cat >/etc/profile.d/bytebase-e2e.sh <<'EOF'
export PATH=/opt/node24/bin:$PATH
export LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8
EOF

step "Versions"
node --version || true
pnpm --version || true
GOTOOLCHAIN="go${GO_VERSION}" go version || true
psql --version || true
ls /root/.cache/ms-playwright 2>/dev/null || true
exit 0
