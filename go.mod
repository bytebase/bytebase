module github.com/bytebase/bytebase

go 1.21.3

require (
	cloud.google.com/go/spanner v1.49.0
	gitee.com/chunanyong/dm v1.8.12
	github.com/ClickHouse/clickhouse-go/v2 v2.14.1
	github.com/antlr4-go/antlr/v4 v4.13.0
	github.com/aws/aws-sdk-go-v2 v1.21.0
	github.com/aws/aws-sdk-go-v2/config v1.18.42
	github.com/aws/aws-sdk-go-v2/credentials v1.13.40
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.87
	github.com/aws/aws-sdk-go-v2/service/s3 v1.40.0
	github.com/blang/semver/v4 v4.0.0
	github.com/bytebase/mysql-parser v0.0.0-20231013095254-61b0903123c6
	github.com/bytebase/plsql-parser v0.0.0-20231016074331-18d9b8d3c031
	github.com/bytebase/postgresql-parser v0.0.0-20230926094140-aa337757cdd0
	github.com/bytebase/snowsql-parser v0.0.0-20230706111031-cafd8faa2dc9
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dgraph-io/ristretto v0.1.1
	github.com/github/gh-ost v1.1.5
	github.com/go-ego/gse v0.80.2
	github.com/go-ldap/ldap/v3 v3.4.6
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/google/cel-go v0.18.1
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.1
	github.com/gosimple/slug v1.13.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/jackc/pgtype v1.14.0
	github.com/jackc/pgx/v4 v4.18.1
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/labstack/echo-contrib v0.15.0
	github.com/labstack/echo/v4 v4.11.1
	github.com/lestrrat-go/jwx/v2 v2.0.12
	github.com/lib/pq v1.10.9
	github.com/lor00x/goldap v0.0.0-20180618054307-a546dffdd1a3
	github.com/mattn/go-sqlite3 v1.14.17
	github.com/microsoft/go-mssqldb v1.6.0
	github.com/nyaruka/phonenumbers v1.1.8
	github.com/paulmach/orb v0.10.0
	github.com/pganalyze/pg_query_go/v4 v4.0.0-00010101000000-000000000000
	github.com/pingcap/tidb v1.1.0-beta.0.20220825063022-5263a0abda61
	github.com/pingcap/tidb/parser v0.0.0-20221101143359-5b0be9af540e
	github.com/pkg/errors v0.9.1
	github.com/redis/go-redis/v9 v9.2.1
	github.com/sashabaranov/go-openai v1.15.4
	github.com/segmentio/analytics-go v3.1.0+incompatible
	github.com/shopspring/decimal v1.3.1
	github.com/sijms/go-ora/v2 v2.7.18
	github.com/snowflakedb/gosnowflake v1.6.25
	github.com/spf13/cobra v1.7.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggo/echo-swagger v1.4.1
	github.com/swaggo/swag v1.16.2
	github.com/tmc/grpc-websocket-proxy v0.0.0-20220101234140-673ab2c3ae75
	github.com/vjeantet/ldapserver v1.0.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	github.com/xo/dburl v0.16.0
	github.com/xuri/excelize/v2 v2.8.0
	go.mongodb.org/mongo-driver v1.12.1
	go.uber.org/multierr v1.11.0
	golang.org/x/crypto v0.13.0
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df
	golang.org/x/oauth2 v0.12.0
	golang.org/x/sys v0.12.0
	golang.org/x/text v0.13.0
	google.golang.org/api v0.143.0
	google.golang.org/genproto v0.0.0-20230920204549-e6e6cdab5c13
	google.golang.org/genproto/googleapis/api v0.0.0-20230920204549-e6e6cdab5c13
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230920204549-e6e6cdab5c13
	google.golang.org/grpc v1.58.2
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.7.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.1.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/JohnCGriffin/overflow v0.0.0-20211019200055-46fa312c352c // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apache/arrow/go/v12 v12.0.1 // indirect
	github.com/apache/thrift v0.19.0 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/danieljoos/wincred v1.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.5.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.5 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.5 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/swaggo/files/v2 v2.0.0 // indirect
	github.com/vcaesar/cedar v0.20.1 // indirect
	github.com/xuri/efp v0.0.0-20230802181842-ad255f2331ca // indirect
	github.com/xuri/nfp v0.0.0-20230819163627-dc951e3ffe1a // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/term v0.12.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	cloud.google.com/go v0.110.8 // indirect
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.2 // indirect
	cloud.google.com/go/longrunning v0.5.1 // indirect
	github.com/BurntSushi/toml v1.2.1 // indirect
	github.com/ClickHouse/ch-go v0.58.2 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.41 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.35 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.43 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.15.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.22.0 // indirect
	github.com/aws/smithy-go v1.14.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/bytebase/tsql-parser v0.0.0-20231019070007-fc13b1c3c56d
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe // indirect
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4 // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/danjacques/gofslock v0.0.0-20220131014315-6e321f4509c8 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/envoyproxy/go-control-plane v0.11.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.2 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-mysql-org/go-mysql v1.6.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/spec v0.20.9 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.1 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgconn v1.14.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.2 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/labstack/gommon v0.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.1 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.4 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20220913051719-115f729f3c8c // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/montanaflynn/stats v0.7.0 // indirect
	github.com/openark/golib v0.0.0-20210531070646-355f37940af8 // indirect
	github.com/opentracing/basictracer-go v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/pingcap/errors v0.11.5-0.20211224045212-9687c2b0f87c // indirect
	github.com/pingcap/failpoint v0.0.0-20220801062533-2eaa32854a6c // indirect
	github.com/pingcap/kvproto v0.0.0-20221101075641-65d0ae8fa853 // indirect
	github.com/pingcap/log v1.1.1-0.20221015072633-39906604fb81 // indirect
	github.com/pingcap/tipb v0.0.0-20221020071514-cd933387bcb5 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20220216144756-c35f1ee13d7c // indirect
	github.com/pquerna/cachecontrol v0.2.0 // indirect
	github.com/pquerna/otp v1.4.0
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.40.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/cors v1.7.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/backo-go v1.0.1 // indirect
	github.com/shirou/gopsutil/v3 v3.22.10 // indirect
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726 // indirect
	github.com/siddontang/go-log v0.0.0-20190221022429-1e957dd83bed // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tikv/client-go/v2 v2.0.1-0.20220725090834-0cdc7c1d0fb9 // indirect
	github.com/tikv/pd/client v0.0.0-20221101140400-25982e60b78a // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.18.0 // indirect
	go.opentelemetry.io/otel/trace v1.18.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

// copied from pingcap/tidb
// fix potential security issue(CVE-2020-26160) introduced by indirect dependency.
replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.6-0.20210809144907-32ab6a8243d7+incompatible

replace github.com/github/gh-ost => github.com/bytebase/gh-ost v1.1.3-0.20230915044519-b37a6525f8f9

replace github.com/pingcap/tidb => github.com/bytebase/tidb v0.0.0-20230914094316-ec1081216cfb

replace github.com/pingcap/tidb/parser => github.com/bytebase/tidb/parser v0.0.0-20230914094316-ec1081216cfb

replace github.com/pganalyze/pg_query_go/v4 => github.com/bytebase/pg_query_go/v4 v4.0.0-20230802100607-2f34e68d96f5
