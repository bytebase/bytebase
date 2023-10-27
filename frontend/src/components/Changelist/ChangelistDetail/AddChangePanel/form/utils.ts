import { orderBy, uniqBy } from "lodash-es";
import { EmptyAffectedTable } from "@/types/changeHistory";
import {
  ChangeHistory,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import { getAffectedTablesOfChangeHistory } from "@/utils";

export const getAffectedTablesFromChangeHistoryList = (
  changeHistoryList: ChangeHistory[]
) => {
  return [
    EmptyAffectedTable,
    ...orderBy(
      uniqBy(
        changeHistoryList
          .map((changeHistory) =>
            getAffectedTablesOfChangeHistory(changeHistory)
          )
          .flat(),
        (affectedTable) => `${affectedTable.schema}.${affectedTable.table}`
      ),
      ["dropped", "table", "schema"]
    ),
  ];
};

export const semanticChangeHistoryType = (type: ChangeHistory_Type) => {
  switch (type) {
    case ChangeHistory_Type.BASELINE:
    case ChangeHistory_Type.MIGRATE:
    case ChangeHistory_Type.MIGRATE_SDL:
    case ChangeHistory_Type.BRANCH:
    case ChangeHistory_Type.MIGRATE_GHOST:
      return ChangeHistory_Type.MIGRATE;
    case ChangeHistory_Type.DATA:
      return ChangeHistory_Type.DATA;
    default:
      return ChangeHistory_Type.UNRECOGNIZED;
  }
};

export const displaySemanticType = (type: ChangeHistory_Type) => {
  const semanticType = semanticChangeHistoryType(type);
  if (semanticType === ChangeHistory_Type.MIGRATE) return "DDL";
  if (semanticType === ChangeHistory_Type.DATA) return "DML";
  return "-";
};
