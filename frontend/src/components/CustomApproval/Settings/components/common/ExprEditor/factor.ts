import { HighLevelFactorList } from "@/plugins/cel";
import { uniq, without } from "lodash-es";

export const NumberFactorList = [
  // Risk related factors
  "affected_rows",
  "level",
  "source",
] as const;

export const StringFactorList = [
  "environment_id", // using `environment.resource_id`
  "project_id", // using `project.resource_id`
  "database_name",
  "db_engine",
  "sql_type",
] as const;

export const FactorList = {
  DDL: uniq([...HighLevelFactorList, ...StringFactorList]),
  DML: uniq([...HighLevelFactorList, ...NumberFactorList, ...StringFactorList]),
  CreateDatabase: without(
    [...HighLevelFactorList, ...StringFactorList],
    "sql_type"
  ),
};
