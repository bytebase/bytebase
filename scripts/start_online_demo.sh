#!/bin/sh

function seedDemoData(){
    echo 'Seeding data for online demo'
    bytebase --host http://localhost --port 80 --demo --data /var/opt/bytebase &
    sleep 10
    echo 'Killing seeding program'
    ps | grep 'bytebase' | grep -v  grep | xargs kill -9
}

function startReadonly(){
    echo 'Initiating readonly mode for online demo'
    bytebase --host http://localhost --port 80 --readonly --demo --data /var/opt/bytebase
}

seedDemoData; startReadonly
