import axios from "axios";
import { defineStore } from "pinia";
import {
  INSTANCE_OPERATION_TIMEOUT,
  QueryInfo,
  ResourceObject,
  SQLResultSet,
  Advice,
  SingleSQLResult,
  Attributes,
} from "@/types";

export function convertSingleSQLResult(
  attributes: Attributes
): SingleSQLResult {
  try {
    return {
      data: JSON.parse((attributes.data as string) || "null"),
      error: attributes.error as string,
    };
  } catch {
    return {
      data: null as any,
      error: attributes.error as string,
    };
  }
}

export function convert(resultSet: ResourceObject): SQLResultSet {
  const resultList: SingleSQLResult[] = [];
  const singleSQLResultAttributesList = resultSet.attributes
    .singleSQLResultList as Attributes[];
  if (Array.isArray(singleSQLResultAttributesList)) {
    singleSQLResultAttributesList.forEach((attributes) => {
      resultList.push(convertSingleSQLResult(attributes));
    });
  }

  return {
    error: (resultSet.attributes.error as string) || "",
    resultList,
    adviceList: resultSet.attributes.adviceList as Advice[],
  };
}

export const useLegacySQLStore = defineStore("legacy_sql", {
  actions: {
    convert(resultSet: ResourceObject): SQLResultSet {
      return convert(resultSet);
    },
    async adminQuery(queryInfo: QueryInfo): Promise<SQLResultSet> {
      const res = (
        await axios.post(
          `/api/sql/execute/admin`,
          {
            data: {
              type: "sqlExecute",
              attributes: {
                ...queryInfo,
                readonly: true,
              },
            },
          },
          {
            timeout: INSTANCE_OPERATION_TIMEOUT,
          }
        )
      ).data;

      const resultSet = convert(res.data);

      return resultSet;
    },
  },
});
