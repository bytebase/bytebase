#!/bin/bash

# For now, we use this script to start our demo on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./demo.sh
# ./demo.sh https://example.com
# ./demo.sh https://example.com:8080
# ./demo.sh https://example.com:8080 schema-migration
# ./demo.sh https://schema-migration.demo.bytebase.com schema-migration

# If no parameter is passed, use https://demo.bytebase.com as host and 443 as port by default
ONLINE_DEMO_EXTERNAL_URL='https://demo.bytebase.com'
ONLINE_DEMO_PORT='443'
if [ $1 ]; then
    PROTOCAL=$(echo $1 | awk -F ':' '{ print $1 }')
    URI=$(echo $1 | awk -F '[/:]' '{ print $4; }')
    PORT=$(echo $1 | awk -F '[/:]' '{ print $5; }')

    ONLINE_DEMO_EXTERNAL_URL=$PROTOCAL://$URI

    if [ $PORT ]; then
        ONLINE_DEMO_PORT=$PORT
    fi
fi

DEMO_NAME='default'
if [ $2 ]; then
    DEMO_NAME=$2
fi

startDemo() {
    echo "Starting Bytebase in demo mode with ${DEMO_NAME} demo on port ${ONLINE_DEMO_PORT}, visiting from ${ONLINE_DEMO_EXTERNAL_URL}..."

    bytebase --port ${ONLINE_DEMO_PORT} --external-url ${ONLINE_DEMO_EXTERNAL_URL} --demo ${DEMO_NAME} --data /var/opt/bytebase
}

startDemo
