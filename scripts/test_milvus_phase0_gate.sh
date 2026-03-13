#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .cache/go-build
export GOCACHE="${ROOT_DIR}/.cache/go-build"

echo "[Phase0] Contract guardrails"
go test -count=1 github.com/bytebase/bytebase/backend/api/v1 -run "^(TestMilvusWiringContract|TestConvertToEngine_AllStoreEnginesMapped|TestConvertEngine_AllV1EnginesMapped)$"
go test -count=1 github.com/bytebase/bytebase/backend/server -run "^(TestUltimateImportsAllDBRegistrationPackages|TestUltimateImportsAllParserRegistrationPackages)$"

echo "[Phase0] Milvus unit + integration baseline"
go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/milvus github.com/bytebase/bytebase/backend/plugin/parser/milvus

echo "[Phase0] Non-Milvus regression sentinels"
# Keep MongoDB sentinel scoped to pure unit tests to avoid external mongosh/testcontainer requirements.
go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/mongodb -run "^(TestGetMongoDBConnectionURL|TestIsMongoStatement|TestGetSimpleStatementResult)$"
go test -count=1 github.com/bytebase/bytebase/backend/plugin/db/elasticsearch

echo "[Phase0] All guardrails passed"
