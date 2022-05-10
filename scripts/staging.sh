#!/bin/sh

# For now, we use this script to start our staging preview on render
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# example usages:
# ./staging.sh
# ./staging.sh https://example.com
# ./staging.sh https://example.com:8080

# If no parameter is passed, use https://staging.bytebase.com as host and 443 as port by default
ONLINE_DEMO_HOST='https://staging.bytebase.com'
ONLINE_DEMO_PORT='443'
if [ $1 ]; then
    PROTOCAL=`echo $1 | awk -F ':' '{ print $1 }'`
    URI=`echo $1 | awk -F '[/:]' '{ print $4; }'`
    PORT=`echo $1 | awk -F '[/:]' '{ print $5; }'`

    ONLINE_DEMO_HOST=$PROTOCAL://$URI

    if [ $PORT ]; then
        ONLINE_DEMO_PORT=$PORT
    fi
fi

echo "Starting Bytebase in demo mode at ${ONLINE_DEMO_HOST}:${ONLINE_DEMO_PORT}..."

bytebase --host ${ONLINE_DEMO_HOST} --port ${ONLINE_DEMO_PORT} --demo --data /var/opt/bytebase --debug
