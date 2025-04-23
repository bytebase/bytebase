import { defineStore } from "pinia";
import { reactive } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import { isValidProjectName, isValidDatabaseName } from "@/types";
import type { QueryHistory } from "@/types/proto/v1/sql_service";

export interface QueryHistoryFilter {
  statement?: string;
  project?: string;
  database?: string;
}

const getListQueryHistoryFilter = (filter: QueryHistoryFilter) => {
  const params = [`type == "QUERY"`];
  if (isValidProjectName(filter.project)) {
    params.push(`project == "${filter.project}"`);
  }
  if (isValidDatabaseName(filter.database)) {
    params.push(`database == "${filter.database}"`);
  }
  if (filter.statement) {
    params.push(`statement.matches("${filter.statement}")`);
  }
  return params.join(" && ");
};

const getCacheKey = (filter: QueryHistoryFilter) => {
  const params = [];
  if (isValidProjectName(filter.project)) {
    params.push(filter.project);
  }
  if (isValidDatabaseName(filter.database)) {
    params.push(filter.database);
  }
  return params.join(".");
};

export const useSQLEditorQueryHistoryStore = defineStore(
  "sqlEditorQueryHistory",
  () => {
    const queryHistoryMap = reactive(
      new Map<
        string,
        { nextPageToken?: string; queryHistories: QueryHistory[] }
      >()
    );

    const fetchQueryHistoryList = async (
      filter: QueryHistoryFilter
    ): Promise<{
      queryHistories: QueryHistory[];
      nextPageToken?: string;
    }> => {
      const key = getCacheKey(filter);
      if (!queryHistoryMap.has(key)) {
        queryHistoryMap.set(key, { queryHistories: [] });
      }
      const pageToken = queryHistoryMap.get(key)?.nextPageToken;

      const resp = await sqlServiceClient.searchQueryHistories({
        pageSize: 5,
        pageToken,
        filter: getListQueryHistoryFilter(filter),
      });

      queryHistoryMap.get(key)!.nextPageToken = resp.nextPageToken;
      if (pageToken) {
        queryHistoryMap.get(key)!.queryHistories.push(...resp.queryHistories);
      } else {
        queryHistoryMap.get(key)!.queryHistories = resp.queryHistories;
      }
      return resp;
    };

    const getQueryHistoryList = (filter: QueryHistoryFilter) => {
      const key = getCacheKey(filter);
      return queryHistoryMap.get(key) ?? { queryHistories: [] };
    };

    const resetPageToken = (filter: QueryHistoryFilter) => {
      const key = getCacheKey(filter);
      if (!queryHistoryMap.has(key)) {
        queryHistoryMap.set(key, { queryHistories: [] });
      }
      queryHistoryMap.get(key)!.nextPageToken = "";
    };

    return {
      resetPageToken,
      fetchQueryHistoryList,
      getQueryHistoryList,
    };
  }
);
