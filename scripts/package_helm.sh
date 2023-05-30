#!/bin/bash

# Exit on error
set -e

# Get current absolute path
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Goto ../helm-charts directory and package the helm chart.
cd $DIR/../helm-charts && helm package bytebase && mv bytebase-*.tgz $DIR/../docs/

# Goto ../docs directory and update the index.yaml
cd $DIR/../docs && helm repo index .
