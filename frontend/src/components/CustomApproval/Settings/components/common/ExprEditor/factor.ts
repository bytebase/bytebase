import { HighLevelFactorList } from "@/plugins/cel";
import { uniq, without } from "lodash-es";

export const NumberFactorList = [
  // Risk related factors
  "affected_rows",
  "level",
  "source",
  "expiration_days",
  "export_rows",
] as const;

export const StringFactorList = [
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "database_name",
  "db_engine",
  "sql_type",
] as const;

export const FactorList = {
  DDL: uniq(
    without(
      [...HighLevelFactorList, ...StringFactorList],
      "expiration_days",
      "export_rows"
    )
  ),
  DML: uniq(
    without(
      [...HighLevelFactorList, ...NumberFactorList, ...StringFactorList],
      "expiration_days",
      "export_rows"
    )
  ),
  CreateDatabase: without(
    [...HighLevelFactorList, ...StringFactorList],
    "sql_type",
    "expiration_days",
    "export_rows"
  ),
  RequestQuery: uniq(
    without(
      [...StringFactorList, ...NumberFactorList],
      "affected_rows",
      "sql_type",
      "export_rows"
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
