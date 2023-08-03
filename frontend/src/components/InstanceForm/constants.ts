import { computed } from "vue";
import { Engine } from "@/types/proto/v1/common";
import { supportedEngineV1List } from "@/utils";

export const defaultPortForEngine = (engine: Engine) => {
  if (engine === Engine.CLICKHOUSE) {
    return "9000";
  } else if (engine === Engine.POSTGRES) {
    return "5432";
  } else if (engine === Engine.SNOWFLAKE) {
    return "443";
  } else if (engine === Engine.TIDB) {
    return "4000";
  } else if (engine === Engine.MONGODB) {
    return "27017";
  } else if (engine === Engine.REDIS) {
    return "6379";
  } else if (engine === Engine.ORACLE) {
    return "1521";
  } else if (engine === Engine.MSSQL) {
    return "1433";
  } else if (engine === Engine.REDSHIFT) {
    return "5439";
  } else if (engine === Engine.OCEANBASE) {
    return "2883";
  } else if (engine === Engine.DM) {
    return "5236";
  }
  return "3306";
};

export const EngineList = computed(() => {
  return supportedEngineV1List();
});

export const EngineIconPath: Record<number, string> = {
  [Engine.MYSQL]: new URL("@/assets/db-mysql.png", import.meta.url).href,
  [Engine.POSTGRES]: new URL("@/assets/db-postgres.png", import.meta.url).href,
  [Engine.TIDB]: new URL("@/assets/db-tidb.png", import.meta.url).href,
  [Engine.SNOWFLAKE]: new URL("@/assets/db-snowflake.png", import.meta.url)
    .href,
  [Engine.CLICKHOUSE]: new URL("@/assets/db-clickhouse.png", import.meta.url)
    .href,
  [Engine.MONGODB]: new URL("@/assets/db-mongodb.png", import.meta.url).href,
  [Engine.SPANNER]: new URL("@/assets/db-spanner.png", import.meta.url).href,
  [Engine.REDIS]: new URL("@/assets/db-redis.png", import.meta.url).href,
  [Engine.ORACLE]: new URL("@/assets/db-oracle.svg", import.meta.url).href,
  [Engine.DM]: new URL("@/assets/db-dm.png", import.meta.url).href,
  [Engine.MSSQL]: new URL("@/assets/db-mssql.svg", import.meta.url).href,
  [Engine.REDSHIFT]: new URL("@/assets/db-redshift.svg", import.meta.url).href,
  [Engine.MARIADB]: new URL("@/assets/db-mariadb.png", import.meta.url).href,
  [Engine.OCEANBASE]: new URL("@/assets/db-oceanbase.png", import.meta.url)
    .href,
};

export const MongoDBConnectionStringSchemaList = [
  "mongodb://",
  "mongodb+srv://",
];

export const SnowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";
