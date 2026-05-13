import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { sqlServiceClientConnect } from "@/connect";
import { isValidDatabaseName, isValidProjectName } from "@/types";
import type { QueryHistory } from "@/types/proto-es/v1/sql_service_pb";
import { SearchQueryHistoriesRequestSchema } from "@/types/proto-es/v1/sql_service_pb";

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
    params.push(`statement.contains("${filter.statement}")`);
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

      const request = create(SearchQueryHistoriesRequestSchema, {
        pageSize: 5,
        pageToken,
        filter: getListQueryHistoryFilter(filter),
      });
      const resp = await sqlServiceClientConnect.searchQueryHistories(request);

      queryHistoryMap.get(key)!.nextPageToken = resp.nextPageToken;
      if (pageToken) {
        // Dedupe by `name` on the append path. The backend uses
        // offset-based page tokens, so when `mergeLatest` prepends a
        // freshly-executed query the cached cursor is now off by one
        // (the new entry shifted every server position). The next
        // "Load more" call may return rows that already live in the
        // cache; filtering by `name` prevents visible duplicates.
        const cachedNames = new Set(
          queryHistoryMap.get(key)!.queryHistories.map((h) => h.name)
        );
        const fresh = resp.queryHistories.filter(
          (h) => !cachedNames.has(h.name)
        );
        queryHistoryMap.get(key)!.queryHistories.push(...fresh);
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
      queryHistoryMap.set(key, { queryHistories: [] });
    };

    /**
     * Post-execute refresh that preserves pagination state. Fetches
     * page 1 (no cursor) and prepends entries whose `name` isn't
     * already cached. Keeps the existing `nextPageToken`, so a user
     * who has loaded pages 1–3 stays at pages 1–3 after running a new
     * query — the just-executed statement appears at the top without
     * resetting the list.
     */
    const mergeLatest = async (filter: QueryHistoryFilter) => {
      const key = getCacheKey(filter);
      const resp = await sqlServiceClientConnect.searchQueryHistories(
        create(SearchQueryHistoriesRequestSchema, {
          pageSize: 5,
          filter: getListQueryHistoryFilter(filter),
        })
      );
      const existing = queryHistoryMap.get(key);
      const existingNames = new Set(
        (existing?.queryHistories ?? []).map((h) => h.name)
      );
      const fresh = resp.queryHistories.filter(
        (h) => !existingNames.has(h.name)
      );
      queryHistoryMap.set(key, {
        nextPageToken: existing?.nextPageToken ?? resp.nextPageToken,
        queryHistories: [...fresh, ...(existing?.queryHistories ?? [])],
      });
      return resp;
    };

    return {
      resetPageToken,
      fetchQueryHistoryList,
      getQueryHistoryList,
      mergeLatest,
    };
  }
);
