module github.com/bytebase/bytebase

go 1.16

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.0.7
	github.com/VictoriaMetrics/fastcache v1.6.0
	github.com/blang/semver/v4 v4.0.0
	github.com/casbin/casbin/v2 v2.40.6
	github.com/github/gh-ost v1.1.4
	github.com/go-sql-driver/mysql v1.6.0
	github.com/golang-jwt/jwt/v4 v4.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/jsonapi v1.0.0
	github.com/google/uuid v1.3.0
	github.com/gosimple/slug v1.10.0
	github.com/labstack/echo/v4 v4.6.1
	github.com/lib/pq v1.10.2
	github.com/mattn/go-sqlite3 v1.14.7
	github.com/pingcap/tidb v1.1.0-beta.0.20211209055157-9f744cdf8266
	github.com/pingcap/tidb/parser v0.0.0-20211209055157-9f744cdf8266
	github.com/pkg/errors v0.9.1
	github.com/qiangmzsx/string-adapter/v2 v2.1.0
	github.com/snowflakedb/gosnowflake v1.6.3
	github.com/spf13/cobra v1.2.0
	github.com/stretchr/testify v1.7.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sys v0.0.0-20220224003255-dbe011f71a99 // indirect
)

// copied from pingcap/tidb
// fix potential security issue(CVE-2020-26160) introduced by indirect dependency.
replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.6-0.20210809144907-32ab6a8243d7+incompatible

replace github.com/github/gh-ost => github.com/bytebase/gh-ost v1.1.4
