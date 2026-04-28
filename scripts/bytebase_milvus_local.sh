#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

BYTEBASE_NAME="${BYTEBASE_CONTAINER_NAME:-bytebase}"
BYTEBASE_IMAGE="${BYTEBASE_IMAGE:-bytebase/bytebase:local-current}"
BYTEBASE_PORT="${BYTEBASE_PORT:-8080}"
BYTEBASE_DATA_DIR="${BYTEBASE_DATA_DIR:-$HOME/.bytebase/data}"
NETWORK_NAME="${BYTEBASE_MILVUS_NETWORK:-bb-net}"
MILVUS_HOST="${MILVUS_DOCKER_HOST:-milvus-standalone}"
MILVUS_PORT="${BYTEBASE_TEST_MILVUS_PORT:-19530}"

usage() {
  cat <<EOF
Usage: $0 <up|down|status|logs|verify>

Commands:
  up      Start Milvus stack, start Bytebase container, wire same network, verify connectivity
  down    Stop Bytebase container (Milvus stack remains)
  status  Show Bytebase + Milvus container status
  logs    Tail Bytebase logs
  verify  Verify Bytebase container can access Milvus API endpoint

Env overrides:
  BYTEBASE_CONTAINER_NAME      (default: bytebase)
  BYTEBASE_IMAGE               (default: bytebase/bytebase:local-current)
  BYTEBASE_PORT                (default: 8080)
  BYTEBASE_DATA_DIR            (default: ~/.bytebase/data)
  BYTEBASE_MILVUS_NETWORK      (default: bb-net)
  MILVUS_DOCKER_HOST           (default: milvus-standalone)
  BYTEBASE_TEST_MILVUS_PORT    (default: 19530)
EOF
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker not found in PATH" >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    echo "docker daemon is not running" >&2
    exit 1
  fi
}

ensure_milvus_up() {
  ./scripts/milvus_docker_local.sh up
}

ensure_network() {
  docker network create "${NETWORK_NAME}" >/dev/null 2>&1 || true
  for c in "${BYTEBASE_NAME}" milvus-standalone milvus-etcd milvus-minio; do
    docker network connect "${NETWORK_NAME}" "${c}" >/dev/null 2>&1 || true
  done
}

wait_bytebase_health() {
  local i code
  for i in $(seq 1 30); do
    code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${BYTEBASE_PORT}/healthz" || true)"
    if [[ "${code}" == "200" ]]; then
      echo "Bytebase is healthy on http://localhost:${BYTEBASE_PORT}"
      return 0
    fi
    sleep 2
  done
  echo "Bytebase did not become healthy on :${BYTEBASE_PORT}" >&2
  docker logs --tail 120 "${BYTEBASE_NAME}" || true
  exit 1
}

cmd_up() {
  require_docker
  ensure_milvus_up

  mkdir -p "${BYTEBASE_DATA_DIR}"
  docker rm -f "${BYTEBASE_NAME}" >/dev/null 2>&1 || true
  docker run -d --init \
    --name "${BYTEBASE_NAME}" \
    --network "${NETWORK_NAME}" \
    -p "${BYTEBASE_PORT}:8080" \
    -v "${BYTEBASE_DATA_DIR}:/var/opt/bytebase" \
    "${BYTEBASE_IMAGE}" \
    --data /var/opt/bytebase --port 8080 >/dev/null

  ensure_network
  wait_bytebase_health
  cmd_verify
}

cmd_down() {
  require_docker
  docker rm -f "${BYTEBASE_NAME}" >/dev/null 2>&1 || true
  echo "Removed container: ${BYTEBASE_NAME}"
}

cmd_status() {
  require_docker
  docker ps -a \
    --filter "name=^/${BYTEBASE_NAME}$" \
    --filter "name=^/milvus-standalone$" \
    --filter "name=^/milvus-etcd$" \
    --filter "name=^/milvus-minio$" \
    --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

cmd_logs() {
  require_docker
  docker logs -f "${BYTEBASE_NAME}"
}

cmd_verify() {
  require_docker
  docker exec "${BYTEBASE_NAME}" sh -lc \
    "curl -sS -m 8 -i -X POST http://${MILVUS_HOST}:${MILVUS_PORT}/v2/vectordb/collections/list -H 'Content-Type: application/json' -d '{}' | head -n 20"
}

main() {
  local cmd="${1:-}"
  case "${cmd}" in
    up) cmd_up ;;
    down) cmd_down ;;
    status) cmd_status ;;
    logs) cmd_logs ;;
    verify) cmd_verify ;;
    *) usage; exit 1 ;;
  esac
}

main "${1:-}"
