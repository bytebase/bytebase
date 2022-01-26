module github.com/bytebase/bytebase

go 1.16

require (
	github.com/ClickHouse/clickhouse-go v1.5.1
	github.com/VictoriaMetrics/fastcache v1.6.0
	github.com/casbin/casbin/v2 v2.40.6
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/jsonapi v1.0.0
	github.com/google/uuid v1.3.0
	github.com/gosimple/slug v1.10.0
	github.com/kr/pretty v0.2.1
	github.com/labstack/echo/v4 v4.6.1
	github.com/lib/pq v1.10.2
	github.com/mattn/go-sqlite3 v2.0.1+incompatible
	github.com/pingcap/parser v0.0.0-20200623164729-3a18f1e5dceb
	github.com/pingcap/tidb v1.1.0-beta.0.20200630082100-328b6d0a955c
	github.com/pkg/errors v0.9.1 // indirect
	github.com/qiangmzsx/string-adapter/v2 v2.1.0
	github.com/snowflakedb/gosnowflake v1.6.3 // indirect
	github.com/spf13/cobra v1.2.0
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5

)

// tidb pulls in the old sqlite3 v2.0.1+incompatible which doesn't support the latest sqlite3 feature such as RETURNING.
// So we use replace to redirect to the desired version v1.14.7.
// FWIW, sqlite3 v2.0.1 is an older version than v1.14.7. The v2 bump is a mistake according to the author.
replace github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.14.7
