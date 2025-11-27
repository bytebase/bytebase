import { computed } from "vue";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { supportedEngineV1List } from "@/utils";

export const defaultPortForEngine = (engine: Engine) => {
  switch (engine) {
    case Engine.CLICKHOUSE:
      return "9000";
    case Engine.MYSQL:
      return "3306";
    case Engine.POSTGRES:
      return "5432";
    case Engine.SNOWFLAKE:
      return "";
    case Engine.SQLITE:
      return "";
    case Engine.TIDB:
      return "4000";
    case Engine.MONGODB:
      return "27017";
    case Engine.REDIS:
      return "6379";
    case Engine.ORACLE:
      return "1521";
    case Engine.SPANNER:
      return "";
    case Engine.MSSQL:
      return "1433";
    case Engine.REDSHIFT:
      return "5439";
    case Engine.MARIADB:
      return "3306";
    case Engine.OCEANBASE:
      return "2883";
    case Engine.STARROCKS:
      return "9030";
    case Engine.DORIS:
      return "9030";
    case Engine.HIVE:
      return "10000";
    case Engine.ELASTICSEARCH:
      return "9200";
    case Engine.BIGQUERY:
      return "";
    case Engine.DYNAMODB:
      return "";
    case Engine.DATABRICKS:
      return "";
    case Engine.COCKROACHDB:
      return "26257";
    case Engine.COSMOSDB:
      return "";
    case Engine.CASSANDRA:
      return "9042";
    case Engine.TRINO:
      return "8080";
  }
  throw new Error("engine port unknown");
};

export const EngineList = computed(() => {
  return supportedEngineV1List();
});

export const EngineIconPath: Record<string, string> = {
  [Engine.MYSQL]: new URL("@/assets/db/mysql.png", import.meta.url).href,
  [Engine.POSTGRES]: new URL("@/assets/db/postgres.png", import.meta.url).href,
  [Engine.TIDB]: new URL("@/assets/db/tidb.png", import.meta.url).href,
  [Engine.SNOWFLAKE]: new URL("@/assets/db/snowflake.png", import.meta.url)
    .href,
  [Engine.CLICKHOUSE]: new URL("@/assets/db/clickhouse.png", import.meta.url)
    .href,
  [Engine.MONGODB]: new URL("@/assets/db/mongodb.png", import.meta.url).href,
  [Engine.SPANNER]: new URL("@/assets/db/spanner.png", import.meta.url).href,
  [Engine.REDIS]: new URL("@/assets/db/redis.png", import.meta.url).href,
  [Engine.ORACLE]: new URL("@/assets/db/oracle.svg", import.meta.url).href,
  [Engine.MSSQL]: new URL("@/assets/db/mssql.svg", import.meta.url).href,
  [Engine.REDSHIFT]: new URL("@/assets/db/redshift.svg", import.meta.url).href,
  [Engine.MARIADB]: new URL("@/assets/db/mariadb.png", import.meta.url).href,
  [Engine.OCEANBASE]: new URL(
    "@/assets/db/oceanbase-mysql.svg",
    import.meta.url
  ).href,
  [Engine.STARROCKS]: new URL("@/assets/db/starrocks.png", import.meta.url)
    .href,
  [Engine.DORIS]: new URL("@/assets/db/doris.png", import.meta.url).href,
  [Engine.HIVE]: new URL("@/assets/db/hive.svg", import.meta.url).href,
  [Engine.ELASTICSEARCH]: new URL(
    "@/assets/db/elasticsearch.svg",
    import.meta.url
  ).href,
  [Engine.BIGQUERY]: new URL("@/assets/db/bigquery.svg", import.meta.url).href,
  [Engine.DYNAMODB]: new URL("@/assets/db/dynamodb.svg", import.meta.url).href,
  [Engine.DATABRICKS]: new URL("@/assets/db/databricks.svg", import.meta.url)
    .href,
  [Engine.COCKROACHDB]: new URL("@/assets/db/cockroachdb.png", import.meta.url)
    .href,
  [Engine.COSMOSDB]: new URL("@/assets/db/cosmosdb.svg", import.meta.url).href,
  [Engine.CASSANDRA]: new URL("@/assets/db/cassandra.svg", import.meta.url)
    .href,
  [Engine.TRINO]: new URL("@/assets/db/trino.svg", import.meta.url).href,
};

export const MongoDBConnectionStringSchemaList = [
  "mongodb://",
  "mongodb+srv://",
];

export const RedisConnectionType = ["Standalone", "Sentinel", "Cluster"];

export const SnowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";
