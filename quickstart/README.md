# Quickstart
This folder stores frequently used databases quickstart.

Prerequest:
- [Docker](https://docs.docker.com/engine/install)

## Clickhouse
1. 
    ```
    git clone https://github.com/bytebase/bytebase.git && \
    cd bytebase/quickstart
    ```

1. Compose up quickstart.
    ```
    docker compose -f clickhouse-quickstart.docker-compose.yml up
    ```
    After services are ready. Open localhost:8080 in brower.

1. `Add instance` and choose `ClickHouse`.
    Click `Test Connection`, then you should see `Successfully connected instance.`.
    
    \[Optional\] Set up `Instance Name` or `Environment`.

    Finially `Create` instance.

1. You should see some test databases have been prepared.
