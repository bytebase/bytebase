#!/bin/sh
cp ../backend/enterprise/plan.yaml ./src/types/
cp ../backend/common/permission/permission.yaml ./src/types/iam/
node ./scripts/generate_permissions.js
