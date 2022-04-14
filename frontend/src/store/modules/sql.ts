import { defineStore } from "pinia";
import axios from "axios";
import {
  ConnectionInfo,
  InstanceId,
  INSTANCE_OPERATION_TIMEOUT,
  QueryInfo,
  ResourceObject,
  SqlResultSet,
} from "@/types";
import { useDatabaseStore } from "./database";
import { useInstanceStore } from "./instance";

function convert(resultSet: ResourceObject): SqlResultSet {
  return {
    data: JSON.parse((resultSet.attributes.data as string) || "{}"),
    error: resultSet.attributes.error as string,
  };
}

export const useSQLStore = defineStore("sql", {
  actions: {
    convert(resultSet: ResourceObject): SqlResultSet {
      return convert(resultSet);
    },

    async ping(connectionInfo: ConnectionInfo) {
      const data = (
        await axios.post(`/api/sql/ping`, {
          data: {
            type: "connectionInfo",
            attributes: connectionInfo,
          },
        })
      ).data.data;

      return convert(data);
    },
    async syncSchema(instanceId: InstanceId) {
      const data = (
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
      ).data.data;

      const resultSet = convert(data);
      if (!resultSet.error) {
        // Refresh the corresponding list.
        useDatabaseStore().fetchDatabaseListByInstanceId(instanceId);
        useInstanceStore().fetchInstanceUserListById(instanceId);
      }

      return resultSet;
    },
    async query(queryInfo: QueryInfo) {
      const data = (
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
      ).data.data;

      const resultSet = convert(data);
      if (resultSet.error) {
        throw new Error(resultSet.error);
      }

      return resultSet.data;
    },
  },
});
