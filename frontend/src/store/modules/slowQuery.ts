import { defineStore } from "pinia";

import { databaseServiceClient, instanceServiceClient } from "@/grpcweb";
import {
  ListSlowQueriesRequest,
  SlowQueryLog,
  Database as V1Database,
} from "@/types/proto/v1/database_service";
import { ComposedSlowQueryLog, Database, unknown, UNKNOWN_ID } from "@/types";
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

  const syncSlowQueriesByInstance = async (instance: string) => {
    await instanceServiceClient.syncSlowQueries({
      instance,
    });
  };

  return { fetchSlowQueryLogList, syncSlowQueriesByInstance };
});

const databaseRequestCache = new Map<string, Promise<V1Database>>();

const getOrFetchV1Database = (name: string) => {
  const cached = databaseRequestCache.get(name);
  if (cached) return cached;
  const request = databaseServiceClient.getDatabase({ name });
  databaseRequestCache.set(name, request);
  return request;
};

const composeSlowQueryLogDatabase = async (
  slowQueryLogList: SlowQueryLog[]
) => {
  const databaseNameList = slowQueryLogList.map((log) => log.resource);
  const databaseIdList = await Promise.all(
    databaseNameList.map((name) => {
      return getOrFetchV1Database(name).then(
        (db) => parseInt(db.uid, 10),
        () => UNKNOWN_ID // fallback for robustness
      );
    })
  );
  const databaseList = await Promise.all(
    databaseIdList.map((id) => {
      if (id === UNKNOWN_ID) return unknown("DATABASE");
      return useDatabaseStore().getOrFetchDatabaseById(id);
    })
  );
  const databaseMap = databaseList.reduce((map, db) => {
    if (db.id !== UNKNOWN_ID) {
      const resource = `instances/${db.instance.resourceId}/databases/${db.name}`;
      map.set(resource, db);
    }
    return map;
  }, new Map<string, Database>());

  return slowQueryLogList.map<ComposedSlowQueryLog>((log) => ({
    log,
    database: databaseMap.get(log.resource) ?? unknown("DATABASE"), // databaseMap.get(log.resource)!,
  }));
};
