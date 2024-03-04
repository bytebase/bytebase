#!/bin/sh
# ===========================================================================
# File: bump_version.sh
# Usage: ./bump_version.sh major.minor.patch
# Description: Bump the version in the $WORKSPACE/scripts/version and $WORKSPACE/helm-charts/bytebase/values.yaml
# ===========================================================================

# exit when any command fails
set -e

# Get current absolute path
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Get the version from the first argument
VERSION=$1

# Failed if the version is not provided
if [ -z "$VERSION" ]; then
    echo "Usage: ./bump_version.sh major.minor.patch"
    exit 1
fi

# Update the version in the $WORKSPACE/scripts/version
echo "Updating version in $DIR/version"
echo $VERSION > $DIR/version

# Update the version in the $WORKSPACE/helm-charts/bytebase/values.yaml
echo "Updating version in $DIR/../helm-charts/bytebase/values.yaml"
sed -i '' "s/version: .*/version: $VERSION/" $DIR/../helm-charts/bytebase/values.yaml

# Running helm lint
echo "Running helm lint"
helm lint $DIR/../helm-charts/bytebase

# Repackage the helm chart
echo "Repackaging the helm chart"
$DIR/package_helm.sh