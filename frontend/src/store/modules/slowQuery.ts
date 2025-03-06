import { defineStore } from "pinia";
import { databaseServiceClient, instanceServiceClient } from "@/grpcweb";
import type { ComposedSlowQueryLog } from "@/types";
import type {
  ListSlowQueriesRequest,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { useDatabaseV1Store, batchGetOrFetchDatabases } from "./v1/database";

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

const composeSlowQueryLogDatabase = async (
  slowQueryLogList: SlowQueryLog[]
) => {
  const databaseNameList = slowQueryLogList.map((log) => log.resource);
  await batchGetOrFetchDatabases(databaseNameList);
  const databaseStore = useDatabaseV1Store();

  return slowQueryLogList.map<ComposedSlowQueryLog>((log, index) => ({
    id: `${index}`,
    log,
    database: databaseStore.getDatabaseByName(log.resource),
  }));
};
