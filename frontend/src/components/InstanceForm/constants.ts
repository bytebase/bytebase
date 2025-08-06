import { computed } from "vue";
import { Engine as OldEngine } from "@/types/proto-es/v1/common_pb";
import { supportedEngineV1List } from "@/utils";

export const defaultPortForEngine = (engine: OldEngine) => {
  switch (engine) {
    case OldEngine.CLICKHOUSE:
      return "9000";
    case OldEngine.MYSQL:
      return "3306";
    case OldEngine.POSTGRES:
      return "5432";
    case OldEngine.SNOWFLAKE:
      return "";
    case OldEngine.SQLITE:
      return "";
    case OldEngine.TIDB:
      return "4000";
    case OldEngine.MONGODB:
      return "27017";
    case OldEngine.REDIS:
      return "6379";
    case OldEngine.ORACLE:
      return "1521";
    case OldEngine.SPANNER:
      return "";
    case OldEngine.MSSQL:
      return "1433";
    case OldEngine.REDSHIFT:
      return "5439";
    case OldEngine.MARIADB:
      return "3306";
    case OldEngine.OCEANBASE:
      return "2883";
    case OldEngine.STARROCKS:
      return "9030";
    case OldEngine.DORIS:
      return "9030";
    case OldEngine.HIVE:
      return "10000";
    case OldEngine.ELASTICSEARCH:
      return "9200";
    case OldEngine.BIGQUERY:
      return "";
    case OldEngine.DYNAMODB:
      return "";
    case OldEngine.DATABRICKS:
      return "";
    case OldEngine.COCKROACHDB:
      return "26257";
    case OldEngine.COSMOSDB:
      return "";
    case OldEngine.CASSANDRA:
      return "9042";
    case OldEngine.TRINO:
      return "8080";
  }
  throw new Error("engine port unknown");
};

export const EngineList = computed(() => {
  return supportedEngineV1List();
});

export const EngineIconPath: Record<string, string> = {
  [OldEngine.MYSQL]: new URL("@/assets/db/mysql.png", import.meta.url).href,
  [OldEngine.POSTGRES]: new URL("@/assets/db/postgres.png", import.meta.url)
    .href,
  [OldEngine.TIDB]: new URL("@/assets/db/tidb.png", import.meta.url).href,
  [OldEngine.SNOWFLAKE]: new URL("@/assets/db/snowflake.png", import.meta.url)
    .href,
  [OldEngine.CLICKHOUSE]: new URL("@/assets/db/clickhouse.png", import.meta.url)
    .href,
  [OldEngine.MONGODB]: new URL("@/assets/db/mongodb.png", import.meta.url).href,
  [OldEngine.SPANNER]: new URL("@/assets/db/spanner.png", import.meta.url).href,
  [OldEngine.REDIS]: new URL("@/assets/db/redis.png", import.meta.url).href,
  [OldEngine.ORACLE]: new URL("@/assets/db/oracle.svg", import.meta.url).href,
  [OldEngine.MSSQL]: new URL("@/assets/db/mssql.svg", import.meta.url).href,
  [OldEngine.REDSHIFT]: new URL("@/assets/db/redshift.svg", import.meta.url)
    .href,
  [OldEngine.MARIADB]: new URL("@/assets/db/mariadb.png", import.meta.url).href,
  [OldEngine.OCEANBASE]: new URL(
    "@/assets/db/oceanbase-mysql.svg",
    import.meta.url
  ).href,
  [OldEngine.STARROCKS]: new URL("@/assets/db/starrocks.png", import.meta.url)
    .href,
  [OldEngine.DORIS]: new URL("@/assets/db/doris.png", import.meta.url).href,
  [OldEngine.HIVE]: new URL("@/assets/db/hive.svg", import.meta.url).href,
  [OldEngine.ELASTICSEARCH]: new URL(
    "@/assets/db/elasticsearch.svg",
    import.meta.url
  ).href,
  [OldEngine.BIGQUERY]: new URL("@/assets/db/bigquery.svg", import.meta.url)
    .href,
  [OldEngine.DYNAMODB]: new URL("@/assets/db/dynamodb.svg", import.meta.url)
    .href,
  [OldEngine.DATABRICKS]: new URL("@/assets/db/databricks.svg", import.meta.url)
    .href,
  [OldEngine.COCKROACHDB]: new URL(
    "@/assets/db/cockroachdb.png",
    import.meta.url
  ).href,
  [OldEngine.COSMOSDB]: new URL("@/assets/db/cosmosdb.svg", import.meta.url)
    .href,
  [OldEngine.CASSANDRA]: new URL("@/assets/db/cassandra.svg", import.meta.url)
    .href,
  [OldEngine.TRINO]: new URL("@/assets/db/trino.svg", import.meta.url).href,
};

export const MongoDBConnectionStringSchemaList = [
  "mongodb://",
  "mongodb+srv://",
];

export const RedisConnectionType = ["Standalone", "Sentinel", "Cluster"];

export const SnowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";
