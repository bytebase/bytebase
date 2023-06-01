import { defineStore } from "pinia";
import axios from "axios";
import {
  DatabaseId,
  InstanceId,
  INSTANCE_OPERATION_TIMEOUT,
  QueryInfo,
  ResourceObject,
  SQLResultSet,
  Advice,
  SingleSQLResult,
  Attributes,
} from "@/types";
import { useLegacyDatabaseStore } from "./database";
import { useDatabaseV1Store } from "./v1";

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

    async syncSchema(instanceId: InstanceId) {
      const res = (
        await axios.post(
          `/api/sql/sync-schema`,
          {
            data: {
              type: "sqlSyncSchema",
              attributes: {
                instanceId: instanceId,
              },
            },
          },
          {
            timeout: INSTANCE_OPERATION_TIMEOUT,
          }
        )
      ).data;

      const resultSet = convert(res.data);
      if (!resultSet.error) {
        // Refresh the corresponding list.
        useLegacyDatabaseStore().fetchDatabaseListByInstanceId(instanceId);
      }

      return resultSet;
    },
    async syncDatabaseSchema(databaseId: DatabaseId) {
      const res = (
        await axios.post(
          `/api/sql/sync-schema`,
          {
            data: {
              type: "sqlSyncSchema",
              attributes: {
                databaseId: databaseId,
              },
            },
          },
          {
            timeout: INSTANCE_OPERATION_TIMEOUT,
          }
        )
      ).data;

      const resultSet = convert(res.data);
      if (!resultSet.error) {
        // Refresh the corresponding list.
        useDatabaseV1Store().fetchDatabaseByUID(String(databaseId));
      }

      return resultSet;
    },
    async query(queryInfo: QueryInfo): Promise<SQLResultSet> {
      const res = (
        await axios.post(
          `/api/sql/execute`,
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
