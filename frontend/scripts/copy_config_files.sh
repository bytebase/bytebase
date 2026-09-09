#!/bin/sh
set -e

# Copy-then-rename rather than a plain cp: the frontend gate refreshes these
# while tsc and the bundler are already reading them, and cp truncates in
# place, so a reader can observe a half-written file. mv within a directory is
# a rename(2) and therefore atomic.
copy_atomic() {
  tmp="$2.$$.tmp"
  cp "$1" "$tmp"
  mv "$tmp" "$2"
}

copy_atomic ../backend/enterprise/plan.yaml ./src/types/plan.yaml
copy_atomic ../backend/common/permission/permission.yaml ./src/types/iam/permission.yaml
node ./scripts/generate_permissions.js
