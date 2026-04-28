#!/usr/bin/env bash

set -euo pipefail

NAME="${MILVUS_CONTAINER_NAME:-milvus-standalone}"
IMAGE="${MILVUS_IMAGE:-milvusdb/milvus:v2.4.15}"
ETCD_NAME="${MILVUS_ETCD_CONTAINER_NAME:-milvus-etcd}"
ETCD_IMAGE="${MILVUS_ETCD_IMAGE:-quay.io/coreos/etcd:v3.5.5}"
MINIO_NAME="${MILVUS_MINIO_CONTAINER_NAME:-milvus-minio}"
MINIO_IMAGE="${MILVUS_MINIO_IMAGE:-minio/minio:RELEASE.2023-03-20T20-16-18Z}"
HOST="${BYTEBASE_TEST_MILVUS_HOST:-127.0.0.1}"
PORT="${BYTEBASE_TEST_MILVUS_PORT:-19530}"
HEALTH_TIMEOUT_SEC="${MILVUS_HEALTH_TIMEOUT_SEC:-180}"

usage() {
  cat <<EOF
Usage: $0 <up|down|status|logs|smoke>

Commands:
  up      Start Milvus standalone in Docker and wait until ready
  down    Stop and remove Milvus stack containers
  status  Show container status
  logs    Tail Milvus logs
  smoke   Run basic operation checks against Milvus HTTP API

Env overrides:
  MILVUS_CONTAINER_NAME   (default: milvus-standalone)
  MILVUS_IMAGE            (default: milvusdb/milvus:v2.4.15)
  MILVUS_ETCD_CONTAINER_NAME (default: milvus-etcd)
  MILVUS_ETCD_IMAGE          (default: quay.io/coreos/etcd:v3.5.5)
  MILVUS_MINIO_CONTAINER_NAME (default: milvus-minio)
  MILVUS_MINIO_IMAGE          (default: minio/minio:RELEASE.2023-03-20T20-16-18Z)
  BYTEBASE_TEST_MILVUS_HOST (default: 127.0.0.1)
  BYTEBASE_TEST_MILVUS_PORT (default: 19530)
  MILVUS_HEALTH_TIMEOUT_SEC (default: 180)
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

wait_ready() {
  local deadline now
  deadline=$((SECONDS + HEALTH_TIMEOUT_SEC))
  echo "Waiting for Milvus at http://${HOST}:${PORT} ..."
  while true; do
    if curl -m 3 -sS "http://${HOST}:${PORT}/api/v1/health" >/dev/null 2>&1; then
      echo "Milvus is ready."
      return 0
    fi
    now=$SECONDS
    if (( now >= deadline )); then
      echo "Timed out waiting for Milvus readiness (${HEALTH_TIMEOUT_SEC}s)." >&2
      echo "Run: $0 logs" >&2
      exit 1
    fi
    sleep 2
  done
}

cmd_up() {
  require_docker
  docker rm -f "${NAME}" "${ETCD_NAME}" "${MINIO_NAME}" >/dev/null 2>&1 || true

  docker run -d \
    --name "${ETCD_NAME}" \
    -p 2379:2379 \
    -e ETCD_AUTO_COMPACTION_MODE=revision \
    -e ETCD_AUTO_COMPACTION_RETENTION=1000 \
    -e ETCD_QUOTA_BACKEND_BYTES=4294967296 \
    -e ETCD_SNAPSHOT_COUNT=50000 \
    "${ETCD_IMAGE}" \
    /usr/local/bin/etcd \
    --advertise-client-urls=http://127.0.0.1:2379 \
    --listen-client-urls=http://0.0.0.0:2379 \
    --data-dir=/etcd >/dev/null

  docker run -d \
    --name "${MINIO_NAME}" \
    -p 9000:9000 \
    -p 9001:9001 \
    -e MINIO_ACCESS_KEY=minioadmin \
    -e MINIO_SECRET_KEY=minioadmin \
    "${MINIO_IMAGE}" \
    minio server /minio_data --console-address ":9001" >/dev/null

  docker rm -f "${NAME}" >/dev/null 2>&1 || true
  docker run -d \
    --name "${NAME}" \
    --link "${ETCD_NAME}:etcd" \
    --link "${MINIO_NAME}:minio" \
    -p "${PORT}:19530" \
    -p 9091:9091 \
    -e ETCD_ENDPOINTS=etcd:2379 \
    -e MINIO_ADDRESS=minio:9000 \
    "${IMAGE}" \
    milvus run standalone >/dev/null
  wait_ready
}

cmd_down() {
  require_docker
  docker rm -f "${NAME}" "${ETCD_NAME}" "${MINIO_NAME}" >/dev/null 2>&1 || true
  echo "Removed containers: ${NAME}, ${ETCD_NAME}, ${MINIO_NAME}."
}

cmd_status() {
  require_docker
  docker ps -a \
    --filter "name=^/${NAME}$" \
    --filter "name=^/${ETCD_NAME}$" \
    --filter "name=^/${MINIO_NAME}$" \
    --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

cmd_logs() {
  require_docker
  echo "==== ${ETCD_NAME} ===="
  docker logs --tail 80 "${ETCD_NAME}" || true
  echo "==== ${MINIO_NAME} ===="
  docker logs --tail 80 "${MINIO_NAME}" || true
  echo "==== ${NAME} ===="
  docker logs --tail 120 "${NAME}" || true
}

cmd_smoke() {
  local col
  col="bb_local_smoke_$(date +%s)"
  echo "Creating collection: ${col}"
  curl -m 8 -sS -X POST "http://${HOST}:${PORT}/v2/vectordb/collections/create" \
    -H 'Content-Type: application/json' \
    -d "{\"collectionName\":\"${col}\",\"dimension\":4}" >/dev/null

  echo "Inserting sample entities"
  curl -m 8 -sS -X POST "http://${HOST}:${PORT}/v2/vectordb/entities/insert" \
    -H 'Content-Type: application/json' \
    -d "{\"collectionName\":\"${col}\",\"data\":[{\"id\":1,\"vector\":[0.1,0.2,0.3,0.4]},{\"id\":2,\"vector\":[0.5,0.6,0.7,0.8]}]}" >/dev/null

  echo "Loading collection"
  curl -m 8 -sS -X POST "http://${HOST}:${PORT}/v2/vectordb/collections/load" \
    -H 'Content-Type: application/json' \
    -d "{\"collectionName\":\"${col}\"}" >/dev/null

  echo "Searching vectors"
  curl -m 8 -sS -X POST "http://${HOST}:${PORT}/v2/vectordb/entities/search" \
    -H 'Content-Type: application/json' \
    -d "{\"collectionName\":\"${col}\",\"data\":[[0.1,0.2,0.3,0.4]],\"annsField\":\"vector\",\"limit\":2,\"outputFields\":[\"id\"]}" \
    | head -n 20

  echo
  echo "Cleaning up collection"
  curl -m 8 -sS -X POST "http://${HOST}:${PORT}/v2/vectordb/collections/drop" \
    -H 'Content-Type: application/json' \
    -d "{\"collectionName\":\"${col}\"}" >/dev/null || true

  echo "Smoke test finished."
}

main() {
  local cmd="${1:-}"
  case "${cmd}" in
    up) cmd_up ;;
    down) cmd_down ;;
    status) cmd_status ;;
    logs) cmd_logs ;;
    smoke) cmd_smoke ;;
    *) usage; exit 1 ;;
  esac
}

main "${1:-}"
