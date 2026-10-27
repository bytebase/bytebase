import { type ReactNode, useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import type { VisualizerEngine } from "@/utils/queryPlanEngines";
import {
  BIGQUERY_PLAN_EMPTY_MESSAGE,
  BIGQUERY_PLAN_INVALID_JSON_MESSAGE,
  BIGQUERY_PLAN_NO_PLAN_MESSAGE,
} from "./bigquery-plan";
import {
  CLICKHOUSE_PLAN_EMPTY_MESSAGE,
  CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE,
  CLICKHOUSE_PLAN_NO_PLAN_MESSAGE,
} from "./clickhouse-plan";
import {
  COCKROACHDB_PLAN_EMPTY_MESSAGE,
  COCKROACHDB_PLAN_INVALID_JSON_MESSAGE,
  COCKROACHDB_PLAN_NO_PLAN_MESSAGE,
} from "./cockroachdb-plan";
import {
  DATABRICKS_PLAN_EMPTY_MESSAGE,
  DATABRICKS_PLAN_NO_PLAN_MESSAGE,
} from "./databricks-plan";
import {
  HIVE_PLAN_EMPTY_MESSAGE,
  HIVE_PLAN_INVALID_JSON_MESSAGE,
  HIVE_PLAN_NO_PLAN_MESSAGE,
} from "./hive-plan";
import {
  MONGODB_PLAN_EMPTY_MESSAGE,
  MONGODB_PLAN_INVALID_JSON_MESSAGE,
  MONGODB_PLAN_NO_PLAN_MESSAGE,
} from "./mongodb-plan";
import {
  MSSQL_PLAN_EMPTY_MESSAGE,
  MSSQL_PLAN_INVALID_XML_MESSAGE,
  MSSQL_PLAN_NO_PLAN_MESSAGE,
} from "./mssql-plan";
import {
  MYSQL_PLAN_EMPTY_MESSAGE,
  MYSQL_PLAN_INVALID_JSON_MESSAGE,
  MYSQL_PLAN_NO_PLAN_MESSAGE,
} from "./mysql-plan";
import {
  OCEANBASE_PLAN_EMPTY_MESSAGE,
  OCEANBASE_PLAN_INVALID_JSON_MESSAGE,
  OCEANBASE_PLAN_NO_PLAN_MESSAGE,
} from "./oceanbase-plan";
import {
  ORACLE_PLAN_EMPTY_MESSAGE,
  ORACLE_PLAN_INVALID_JSON_MESSAGE,
  ORACLE_PLAN_NO_PLAN_MESSAGE,
} from "./oracle-plan";
import {
  QueryPlanI18nProvider,
  type QueryPlanTranslate,
  type QueryPlanTranslationKey,
  useQueryPlanTranslation,
} from "./plan-i18n";
import {
  PLAN_TOO_DEEP_MESSAGE,
  PLAN_TOO_MANY_NODES_MESSAGE,
  type PlanTableLookup,
} from "./plan-model";
import { parsePlan } from "./plan-parsers";
import {
  POSTGRES_PLAN_EMPTY_MESSAGE,
  POSTGRES_PLAN_INVALID_JSON_MESSAGE,
  POSTGRES_PLAN_NO_PLAN_MESSAGE,
} from "./postgres-plan";
import { QueryPlanViewer } from "./QueryPlanViewer";
import {
  REDSHIFT_PLAN_EMPTY_MESSAGE,
  REDSHIFT_PLAN_NO_PLAN_MESSAGE,
} from "./redshift-plan";
import {
  SNOWFLAKE_PLAN_EMPTY_MESSAGE,
  SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE,
  SNOWFLAKE_PLAN_NO_PLAN_MESSAGE,
} from "./snowflake-plan";
import {
  SPANNER_PLAN_EMPTY_MESSAGE,
  SPANNER_PLAN_EMULATOR_MESSAGE,
  SPANNER_PLAN_INVALID_JSON_MESSAGE,
  SPANNER_PLAN_NO_PLAN_MESSAGE,
} from "./spanner-plan";
import {
  STARROCKS_PLAN_EMPTY_MESSAGE,
  STARROCKS_PLAN_NO_PLAN_MESSAGE,
} from "./starrocks-plan";
import {
  TIDB_PLAN_EMPTY_MESSAGE,
  TIDB_PLAN_INVALID_JSON_MESSAGE,
  TIDB_PLAN_NO_PLAN_MESSAGE,
} from "./tidb-plan";
import {
  TRINO_PLAN_EMPTY_MESSAGE,
  TRINO_PLAN_INVALID_JSON_MESSAGE,
  TRINO_PLAN_NO_PLAN_MESSAGE,
} from "./trino-plan";

const STATUS_CLASS =
  "flex h-full min-h-0 w-full flex-col gap-4 overflow-auto bg-background p-4";

const PARSE_ERROR_KEYS = new Map<string, QueryPlanTranslationKey>([
  [PLAN_TOO_DEEP_MESSAGE, "error.too-deep"],
  [PLAN_TOO_MANY_NODES_MESSAGE, "error.too-many-nodes"],
  [POSTGRES_PLAN_EMPTY_MESSAGE, "error.postgres-empty"],
  [POSTGRES_PLAN_INVALID_JSON_MESSAGE, "error.postgres-invalid"],
  [POSTGRES_PLAN_NO_PLAN_MESSAGE, "error.postgres-no-plan"],
  [MSSQL_PLAN_EMPTY_MESSAGE, "error.mssql-empty"],
  [MSSQL_PLAN_INVALID_XML_MESSAGE, "error.mssql-invalid"],
  [MSSQL_PLAN_NO_PLAN_MESSAGE, "error.mssql-no-plan"],
  [SPANNER_PLAN_EMPTY_MESSAGE, "error.spanner-empty"],
  [SPANNER_PLAN_INVALID_JSON_MESSAGE, "error.spanner-invalid"],
  [SPANNER_PLAN_NO_PLAN_MESSAGE, "error.spanner-no-plan"],
  [SPANNER_PLAN_EMULATOR_MESSAGE, "error.spanner-emulator"],
  [MYSQL_PLAN_EMPTY_MESSAGE, "error.mysql-empty"],
  [MYSQL_PLAN_INVALID_JSON_MESSAGE, "error.mysql-invalid"],
  [MYSQL_PLAN_NO_PLAN_MESSAGE, "error.mysql-no-plan"],
  [TIDB_PLAN_EMPTY_MESSAGE, "error.tidb-empty"],
  [TIDB_PLAN_INVALID_JSON_MESSAGE, "error.tidb-invalid"],
  [TIDB_PLAN_NO_PLAN_MESSAGE, "error.tidb-no-plan"],
  [ORACLE_PLAN_EMPTY_MESSAGE, "error.oracle-empty"],
  [ORACLE_PLAN_INVALID_JSON_MESSAGE, "error.oracle-invalid"],
  [ORACLE_PLAN_NO_PLAN_MESSAGE, "error.oracle-no-plan"],
  [BIGQUERY_PLAN_EMPTY_MESSAGE, "error.bigquery-empty"],
  [BIGQUERY_PLAN_INVALID_JSON_MESSAGE, "error.bigquery-invalid"],
  [BIGQUERY_PLAN_NO_PLAN_MESSAGE, "error.bigquery-no-plan"],
  [CLICKHOUSE_PLAN_EMPTY_MESSAGE, "error.clickhouse-empty"],
  [CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE, "error.clickhouse-invalid"],
  [CLICKHOUSE_PLAN_NO_PLAN_MESSAGE, "error.clickhouse-no-plan"],
  [TRINO_PLAN_EMPTY_MESSAGE, "error.trino-empty"],
  [TRINO_PLAN_INVALID_JSON_MESSAGE, "error.trino-invalid"],
  [TRINO_PLAN_NO_PLAN_MESSAGE, "error.trino-no-plan"],
  [OCEANBASE_PLAN_EMPTY_MESSAGE, "error.oceanbase-empty"],
  [OCEANBASE_PLAN_INVALID_JSON_MESSAGE, "error.oceanbase-invalid"],
  [OCEANBASE_PLAN_NO_PLAN_MESSAGE, "error.oceanbase-no-plan"],
  [MONGODB_PLAN_EMPTY_MESSAGE, "error.mongodb-empty"],
  [MONGODB_PLAN_INVALID_JSON_MESSAGE, "error.mongodb-invalid"],
  [MONGODB_PLAN_NO_PLAN_MESSAGE, "error.mongodb-no-plan"],
  [SNOWFLAKE_PLAN_EMPTY_MESSAGE, "error.snowflake-empty"],
  [SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE, "error.snowflake-invalid"],
  [SNOWFLAKE_PLAN_NO_PLAN_MESSAGE, "error.snowflake-no-plan"],
  [HIVE_PLAN_EMPTY_MESSAGE, "error.hive-empty"],
  [HIVE_PLAN_INVALID_JSON_MESSAGE, "error.hive-invalid"],
  [HIVE_PLAN_NO_PLAN_MESSAGE, "error.hive-no-plan"],
  [DATABRICKS_PLAN_EMPTY_MESSAGE, "error.databricks-empty"],
  [DATABRICKS_PLAN_NO_PLAN_MESSAGE, "error.databricks-no-plan"],
  [COCKROACHDB_PLAN_EMPTY_MESSAGE, "error.cockroachdb-empty"],
  [COCKROACHDB_PLAN_INVALID_JSON_MESSAGE, "error.cockroachdb-invalid"],
  [COCKROACHDB_PLAN_NO_PLAN_MESSAGE, "error.cockroachdb-no-plan"],
  [REDSHIFT_PLAN_EMPTY_MESSAGE, "error.redshift-empty"],
  [REDSHIFT_PLAN_NO_PLAN_MESSAGE, "error.redshift-no-plan"],
  [STARROCKS_PLAN_EMPTY_MESSAGE, "error.starrocks-empty"],
  [STARROCKS_PLAN_NO_PLAN_MESSAGE, "error.starrocks-no-plan"],
]);

interface Props {
  engine: VisualizerEngine;
  planSource: string;
  textPlanSource?: string;
  note?: ReactNode;
  describeTable?: PlanTableLookup;
  translate?: QueryPlanTranslate;
  disallowCopyingData?: boolean;
}

function QueryPlanViewContent({
  engine,
  planSource,
  textPlanSource,
  note,
  describeTable,
  disallowCopyingData = false,
}: Props) {
  const translate = useQueryPlanTranslation();
  const result = useMemo(
    () => parsePlan(engine, planSource),
    [engine, planSource]
  );

  if (!result.ok) {
    const errorKey = PARSE_ERROR_KEYS.get(result.message);
    const fallbackPlan = textPlanSource?.trim() ? textPlanSource : planSource;
    return (
      <div className={cn(STATUS_CLASS, disallowCopyingData && "select-none")}>
        <Alert
          variant="error"
          title={translate("error.unreadable")}
          description={errorKey ? translate(errorKey) : result.message}
        />
        {fallbackPlan.trim() ? (
          <pre className="font-mono text-xs leading-4 break-words whitespace-pre-wrap text-control">
            {fallbackPlan}
          </pre>
        ) : null}
      </div>
    );
  }

  return (
    <QueryPlanViewer
      tree={result.tree}
      rawPlan={planSource}
      textPlan={textPlanSource}
      note={note}
      describeTable={describeTable}
      disallowCopyingData={disallowCopyingData}
    />
  );
}

export function QueryPlanView({ translate, ...props }: Props) {
  return (
    <QueryPlanI18nProvider translate={translate}>
      <QueryPlanViewContent {...props} />
    </QueryPlanI18nProvider>
  );
}
