#! /bin/sh
n=4
r=2
for i in $(seq 1 $n); do
    REPLICA=$(printf "%02d" $i)
    SHARD=$(printf "%02d" $(((i-1) % r + 1)))
    DIR=$(dirname $0)
    REPLICA=$REPLICA SHARD=$SHARD envsubst < $DIR/config.xml > $DIR/clickhouse$REPLICA.config.xml
done
