import { useEnvironmentV1Store } from "@/store";
import { uniq } from "lodash-es";

import { LABEL_VALUE_EMPTY } from "../label";
import { extractEnvironmentResourceName } from "./environment";
import { Database } from "@/types/proto/v1/database_service";

export const getLabelValueFromLabelListV1 = (
  labels: Record<string, string>,
  key: string
) => {
  return labels[key] ?? LABEL_VALUE_EMPTY;
};

export const getLabelValueV1 = (db: Database, key: string): string => {
  return getLabelValueFromLabelListV1(db.labels, key);
};

export const getLabelValuesFromDatabaseV1List = (
  key: string,
  databaseList: Database[],
  withEmptyValue = false
): string[] => {
  if (key === "bb.environment") {
    const environmentList = useEnvironmentV1Store().getEnvironmentList();
    return environmentList.map((env) =>
      extractEnvironmentResourceName(env.name)
    );
  }

  const valueList = databaseList.flatMap((db) => {
    if (key in db.labels) {
      return [db.labels[key]];
    }
    return [];
  });
  // Select all distinct database label values of {{key}}
  const distinctValueList = uniq(valueList);

  if (withEmptyValue) {
    // plus one more "<empty value>" if needed
    distinctValueList.push(LABEL_VALUE_EMPTY);
  }

  return distinctValueList;
};
