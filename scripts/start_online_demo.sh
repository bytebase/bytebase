#!/bin/sh

function seedData(){
    echo 'Seeding data for online demo'
    bytebase --host http://localhost --port 80 --demo --data /var/opt/bytebase >& /dev/null &
    sleep 5
    ps | grep 'bytebase --demo' | grep -v  grep | xargs kill -9
}

function startReadonly(){
    echo 'Initiating readonly mode for online demo'
    bytebase --host http://localhost --port 80  --readonly --data /var/opt/bytebase
}

seedData; startReadonly
