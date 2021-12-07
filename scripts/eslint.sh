#!/bin/sh

if which node > /dev/null; then
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

    cd frontend && yarn && npm run lint:staged
else
    echo 'Node not installed'
    exit 0
fi
