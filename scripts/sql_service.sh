#!/bin/bash

# For now, we use this script to start our SQL service on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./sql_service.sh
# ./sql_service.sh --host https://example.com
# ./sql_service.sh --host https://example.com --port 8080 --workspace-id bytebase-sql-service

ONLINE_HOST='http://localhost'
ONLINE_PORT='80'
WORKSPACE_ID=''

# Get parameters
for i in "$@"
do
case $i in
    --host)
    ONLINE_HOST="$2"
    shift
    ;;
    --port)
    ONLINE_PORT="$2"
    shift
    ;;
    --workspace-id)
    WORKSPACE_ID="$2"
    shift # past argument
    ;;
    *) # unknown option
    ;;
esac
done

echo "Starting Bytebase SQL Service at ${ONLINE_HOST}:${ONLINE_PORT} with workspace ${WORKSPACE_ID}..."

# Start the SQL service with workspace id
sql-service --host ${ONLINE_HOST} --port ${ONLINE_PORT} --workspace-id ${WORKSPACE_ID}
