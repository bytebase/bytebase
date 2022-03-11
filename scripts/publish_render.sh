#!/bin/sh

# For now, we use this script to start our service on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./publish_render.sh

# If no parameter is passed, use xxxx as host and 80 as port by default
HOST='https://localhost'
PORT='80'
if [ $1 ]; then
    PROTOCAL=`echo $1 | awk -F ':' '{ print $1 }'`
    URI=`echo $1 | awk -F '[/:]' '{ print $4; }'`
    PORT=`echo $1 | awk -F '[/:]' '{ print $5; }'`

    HOST=$PROTOCAL://$URI

    if [ $PORT ]; then
        PORT=$PORT
    fi
fi

echo "Starting Bytebase at ${HOST}:${PORT}..."

bytebase --host ${HOST} --port ${PORT} --data /var/opt/bytebase
