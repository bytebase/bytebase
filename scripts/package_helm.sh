#!/bin/bash

# Exit on error
set -e

# Get current absolute path
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

cd $DIR/../helm-charts

mkdir -p new_charts
helm package -u -d new_charts bytebase
helm repo index --merge $DIR/../docs/index.yaml new_charts

# Update charts repo.
mv new_charts/*.tgz $DIR/../docs
mv new_charts/index.yaml $DIR/../docs/index.yaml
rm -rf new_charts
