import { defineStore } from "pinia";
import { databaseServiceClient, instanceServiceClient } from "@/grpcweb";
import type { ComposedSlowQueryLog, ComposedDatabase } from "@/types";
import type {
  ListSlowQueriesRequest,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { useDatabaseV1Store } from "./v1/database";

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
    } catch {
      return [];
    }
  };

  const syncSlowQueries = async (parent: string) => {
    await instanceServiceClient.syncSlowQueries({
      parent,
    });
  };

  return { fetchSlowQueryLogList, syncSlowQueries };
});

const databaseRequestCache = new Map<string, Promise<ComposedDatabase>>();

const getOrFetchV1Database = (name: string) => {
  const cached = databaseRequestCache.get(name);
  if (cached) return cached;
  const request = useDatabaseV1Store().getOrFetchDatabaseByName(name);
  databaseRequestCache.set(name, request);
  return request;
};

const composeSlowQueryLogDatabase = async (
  slowQueryLogList: SlowQueryLog[]
) => {
  const databaseNameList = slowQueryLogList.map((log) => log.resource);
  await Promise.all(
    databaseNameList.map((name) => {
      return getOrFetchV1Database(name);
    })
  );
  return slowQueryLogList.map<ComposedSlowQueryLog>((log, index) => ({
    id: `${index}`,
    log,
    database: useDatabaseV1Store().getDatabaseByName(log.resource),
  }));
};
