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

seedDemoData() {
    echo 'Seeding data for online demo'

    bytebase --port ${ONLINE_DEMO_PORT} --external-url ${ONLINE_DEMO_EXTERNAL_URL} --demo ${DEMO_NAME} --data /var/opt/bytebase &

    until [ -f /var/opt/bytebase/pgdata-demo/default/PG_VERSION ]; do
        echo "waiting..."
        sleep 1
    done
    echo 'Sleep 120 seconds for Bytebase to finish migration...'
    sleep 120

    echo 'Killing seeding program'

    killall bytebase

    sleep 20
}

startReadonly() {
    echo "Starting Bytebase in readonly and demo mode on port ${ONLINE_DEMO_PORT}, visiting from ${ONLINE_DEMO_EXTERNAL_URL}..."

    bytebase --port ${ONLINE_DEMO_PORT} --external-url ${ONLINE_DEMO_EXTERNAL_URL} --readonly --demo ${DEMO_NAME} --data /var/opt/bytebase
}

seedDemoData
startReadonly
