# Quickstart
This folder stores frequently used databases quickstart.

Prerequisite:
- [Docker](https://docs.docker.com/engine/install)

## MySQL
1. 
    ```
    git clone https://github.com/bytebase/bytebase.git && \
    cd bytebase/quickstart
    ```

1. Compose up quickstart.
    ```
    docker compose -f mysql-quickstart.docker-compose.yml up
    ```
    After services are ready. Open localhost:8080 in brower.

1. In `Instances`, there are 2 prepared MySQL instances both connect to `host.docker.internal:3306`. Choose `MySQL Test`.

    Find `Connection info`, check `Empty` password, and `Test Connection`. You should see `Successfully connected instance.`.
    `Update` password.

    Next `Create migration schema` (in the top), and choose `Create`.

    Finally `Sync Now` (in the bottom). You should see some test databases.

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

    Finially `Create` instance. You should see some test databases have been prepared.
