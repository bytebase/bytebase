import { HighLevelFactorList } from "@/plugins/cel";
import { uniq, without } from "lodash-es";

export const NumberFactorList = [
  // Risk related factors
  "affected_rows",
  "level",
  "source",
  "expiration_day",
  "export_row",
] as const;

export const StringFactorList = [
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "database_name",
  "table_name",
  "db_engine",
  "sql_type",
] as const;

export const FactorList = {
  DDL: uniq(
    without(
      [...HighLevelFactorList, ...StringFactorList],
      "expiration_day",
      "export_row",
      "table_name"
    )
  ),
  DML: uniq(
    without(
      [...HighLevelFactorList, ...NumberFactorList, ...StringFactorList],
      "expiration_day",
      "export_row",
      "table_name"
    )
  ),
  CreateDatabase: without(
    [...HighLevelFactorList, ...StringFactorList],
    "sql_type",
    "expiration_day",
    "export_row",
    "table_name"
  ),
  RequestQuery: uniq(
    without(
      [...StringFactorList, ...NumberFactorList],
      "affected_rows",
      "sql_type",
      "export_row"
    )
  ),
  RequestExport: uniq(
    without(
      [...StringFactorList, ...NumberFactorList],
      "affected_rows",
      "sql_type"
    )
  ),
};
