#!/bin/bash

# For now, we use this script to start our staging preview on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./staging.sh
# ./staging.sh https://example.com
# ./staging.sh https://example.com:8080
# ./staging.sh https://staging.bytebase.com postgres://user:secret@postgres.example.com/bytebase

# If no parameter is passed, use https://staging.bytebase.com as host and 443 as port by default
ONLINE_DEMO_EXTERNAL_URL='https://staging.bytebase.com'
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

PG=''
if [ $2 ]; then
    PG="--pg $2"
fi

echo "Starting Bytebase in demo mode on port ${ONLINE_DEMO_PORT}, visiting from ${ONLINE_DEMO_EXTERNAL_URL} ..."

bytebase --port ${ONLINE_DEMO_PORT} --external-url ${ONLINE_DEMO_EXTERNAL_URL} ${PG} --data /var/opt/bytebase --demo default --debug
