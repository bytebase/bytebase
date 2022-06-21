import { defineStore } from "pinia";
import axios from "axios";
import {
  ConnectionInfo,
  InstanceId,
  INSTANCE_OPERATION_TIMEOUT,
  QueryInfo,
  ResourceObject,
  SQLResultSet,
  Advice,
} from "@/types";
import { useDatabaseStore } from "./database";
import { useInstanceStore } from "./instance";

function convert(resultSet: ResourceObject): SQLResultSet {
  return {
    data: JSON.parse((resultSet.attributes.data as string) || "null"),
    error: resultSet.attributes.error as string,
    adviceList: resultSet.attributes.adviceList as Advice[],
  };
}

export const useSQLStore = defineStore("sql", {
  actions: {
    convert(resultSet: ResourceObject): SQLResultSet {
      return convert(resultSet);
    },

    async ping(connectionInfo: ConnectionInfo) {
      const res = (
        await axios.post(`/api/sql/ping`, {
          data: {
            type: "connectionInfo",
            attributes: connectionInfo,
          },
        })
      ).data;

      return convert(res.data);
    },
    async syncSchema(instanceId: InstanceId) {
      const res = (
        await axios.post(
          `/api/sql/sync-schema`,
          {
            data: {
              type: "sqlSyncSchema",
              attributes: {
                instanceId,
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
        useDatabaseStore().fetchDatabaseListByInstanceId(instanceId);
        useInstanceStore().fetchInstanceUserListById(instanceId);
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
      if (resultSet.error) {
        throw new Error(resultSet.error);
      }

      return resultSet;
    },
  },
});
