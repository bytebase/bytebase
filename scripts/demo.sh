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
ONLINE_DEMO_HOST='https://demo.bytebase.com'
ONLINE_DEMO_PORT='443'
if [ $1 ]; then
    PROTOCAL=$(echo $1 | awk -F ':' '{ print $1 }')
    URI=$(echo $1 | awk -F '[/:]' '{ print $4; }')
    PORT=$(echo $1 | awk -F '[/:]' '{ print $5; }')

    ONLINE_DEMO_HOST=$PROTOCAL://$URI

    if [ $PORT ]; then
        ONLINE_DEMO_PORT=$PORT
    fi
fi

DEMO_NAME=''
if [ $2 ]; then
    DEMO_NAME=$2
fi

seedDemoData() {
    echo 'Seeding data for online demo'

    bytebase --host ${ONLINE_DEMO_HOST} --port ${ONLINE_DEMO_PORT} --demo --demo-name ${DEMO_NAME} --data /var/opt/bytebase &

    until [ -f /var/opt/bytebase/pgdata/PG_VERSION ]; do
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
    echo "Starting Bytebase in readonly and demo mode at ${ONLINE_DEMO_HOST}:${ONLINE_DEMO_PORT}..."

    bytebase --host ${ONLINE_DEMO_HOST} --port ${ONLINE_DEMO_PORT} --readonly --demo -demo-name ${DEMO_NAME} --data /var/opt/bytebase
}

seedDemoData
startReadonly
