import { defineStore } from "pinia";
import { uniq } from "lodash-es";

import { databaseServiceClient, instanceServiceClient } from "@/grpcweb";
import {
  ListSlowQueriesRequest,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { ComposedSlowQueryLog, Instance, unknown } from "@/types";
import { useDatabaseStore } from "./database";

export const useSlowQueryStore = defineStore("slow-query", () => {
  const fetchSlowQueryLogList = async (
    request: Partial<ListSlowQueriesRequest> = {}
  ) => {
    try {
      const response = await databaseServiceClient.listSlowQueries(request);
      const composedLogList = await composeSlowQueryLogDatabase(
        response.slowQueryLogs
      );
      return composedLogList;
    } catch (ex) {
      return [];
    }
  };

  const syncSlowQueriesByInstance = async (instance: Instance) => {
    const name = `environments/${instance.environment.resourceId}/instances/${instance.resourceId}`;
    await instanceServiceClient.syncSlowQueries({
      instance: name,
    });
  };

  return { fetchSlowQueryLogList, syncSlowQueriesByInstance };
});

const composeSlowQueryLogDatabase = async (
  slowQueryLogList: SlowQueryLog[]
) => {
  const databaseNameList = uniq(slowQueryLogList.map((log) => log.resource));
  const databaseIdList = await Promise.all(
    databaseNameList.map((name) => {
      return databaseServiceClient.getDatabase({ name }).then((db) => db.uid);
    })
  );
  const databaseList = await Promise.all(
    databaseIdList.map((id) => {
      return useDatabaseStore().fetchDatabaseById(id);
    })
  );
  const databaseMap = new Map(
    databaseList.map((db) => [
      `environments/${db.instance.environment.resourceId}/instances/${db.instance.resourceId}/databases/${db.name}`,
      db,
    ])
  );

  return slowQueryLogList.map<ComposedSlowQueryLog>((log) => ({
    log,
    database: databaseMap.get(log.resource) ?? unknown("DATABASE"), // databaseMap.get(log.resource)!,
  }));
};
