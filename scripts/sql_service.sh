#!/bin/bash

# Start the SQL service with workspace id

ONLINE_HOST='http://localhost'
ONLINE_PORT='80'
WORKSPACE_ID=''

# Get parameters
for i in "$@"
do
case $i in
    --host=*)
    ONLINE_HOST="${i#*=}"
    shift
    ;;
    --port=*)
    ONLINE_PORT="${i#*=}"
    shift
    ;;
    --workspace-id=*)
    WORKSPACE_ID="${i#*=}"
    shift
    ;;
    *) # unknown option
    ;;
esac
done

echo "Starting Bytebase SQL Service at ${ONLINE_HOST}:${ONLINE_PORT} with workspace ${WORKSPACE_ID}..."

sql-service --host ${ONLINE_HOST} --port ${ONLINE_PORT} --workspace-id ${WORKSPACE_ID}
