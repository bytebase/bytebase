#!/bin/bash

# For now, we use this script to start our sql service on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./sql_service.sh
# ./sql_service.sh https://example.com
# ./sql_service.sh https://example.com:8080

# If no parameter is passed, use https://sql.bytebase.com as host and 443 as port by default
ONLINE_HOST='https://sql.bytebase.com'
ONLINE_PORT='443'
if [ $1 ]; then
    PROTOCAL=$(echo $1 | awk -F ':' '{ print $1 }')
    URI=$(echo $1 | awk -F '[/:]' '{ print $4; }')
    PORT=$(echo $1 | awk -F '[/:]' '{ print $5; }')

    ONLINE_HOST=$PROTOCAL://$URI

    if [ $PORT ]; then
        ONLINE_PORT=$PORT
    fi
fi

echo "Starting Bytebase SQL Service in debug mode at ${ONLINE_HOST}:${ONLINE_PORT}..."

sql-service --host ${ONLINE_HOST} --port ${ONLINE_PORT} --debug --data /var/opt/sql-service
