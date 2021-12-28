#!/bin/sh

function seedData(){
    echo 'Seeding data for online demo'
    bytebase --host http://localhost --port 80 --demo --data /var/opt/bytebase >& /dev/null &
    sleep 10
    echo 'Killing seeding program'
    ps | grep 'bytebase' | grep -v  grep | xargs kill -9
}

function startReadonly(){
    echo 'Initiating readonly mode for online demo'
    bytebase --host http://localhost --port 80  --readonly --data /var/opt/bytebase
}

seedData; startReadonly
