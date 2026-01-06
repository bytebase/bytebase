#!/bin/sh
cp ../backend/enterprise/plan.yaml ./src/types/
cp ../backend/component/iam/permission.yaml ./src/types/iam/
node ./scripts/generate_permissions.js
