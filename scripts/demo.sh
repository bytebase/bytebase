#!/bin/sh

# For now, we use this script to start our demo on rederer
# by changing the ENTRYPOINT and CMD at the dockerfile to this.

# If no parameter is passed, use https://demo.bytebase.com:80 as host by default
ONLINE_DEMO_HOST=$1
ONLINE_DEMO_PORT=$2
if [ ! $ONLINE_DEMO_HOST ];then
    ONLINE_DEMO_HOST='https://demo.bytebase.com'
fi
if [ ! $ONLINE_DEMO_PORT ];then
    ONLINE_DEMO_PORT='80'
fi

function seedDemoData(){
    echo 'Seeding data for online demo'
    
    bytebase --host ${ONLINE_DEMO_HOST} --port ${ONLINE_DEMO_PORT} --demo --data /var/opt/bytebase &
    
    echo 'Sleep 10 seconds for bytebase to finish migration...'
    
    sleep 10

    echo 'Killing seeding program'
    
    ps | grep 'bytebase'  | grep -v grep | xargs kill -9
}

function startReadonly(){
    echo 'Starting Bytebase in readonly and demo mode.'
    
    bytebase --host ${ONLINE_DEMO_HOST} --port ${ONLINE_DEMO_PORT} --readonly --demo --data /var/opt/bytebase
}

seedDemoData; startReadonly
