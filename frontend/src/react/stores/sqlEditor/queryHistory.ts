import { create } from "@bufbuild/protobuf";
import { sqlServiceClientConnect } from "@/connect";
import type { QueryHistory } from "@/types/proto-es/v1/sql_service_pb";
import { SearchQueryHistoriesRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import { isValidProjectName } from "@/types/v1/project";
import type {
  QueryHistoryEntry,
  QueryHistoryFilter,
  QueryHistorySlice,
  SQLEditorSliceCreator,
  SQLEditorStoreState,
} from "./types";

const EMPTY_ENTRY: QueryHistoryEntry = { queryHistories: [] };

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

export const getQueryHistoryCacheKey = (filter: QueryHistoryFilter): string => {
  const params: string[] = [];
  if (isValidProjectName(filter.project)) {
    params.push(filter.project as string);
  }
  if (isValidDatabaseName(filter.database)) {
    params.push(filter.database as string);
  }
  return params.join(".");
};

export const createQueryHistorySlice: SQLEditorSliceCreator<
  QueryHistorySlice
> = (set, get) => ({
  queryHistoryByKey: {},

  fetchQueryHistoryList: async (filter) => {
    const key = getQueryHistoryCacheKey(filter);
    const existing = get().queryHistoryByKey[key] ?? EMPTY_ENTRY;
    const pageToken = existing.nextPageToken;

    const request = create(SearchQueryHistoriesRequestSchema, {
      pageSize: 5,
      pageToken,
      filter: getListQueryHistoryFilter(filter),
    });
    const resp = await sqlServiceClientConnect.searchQueryHistories(request);

    set((s) => {
      const prev = s.queryHistoryByKey[key] ?? EMPTY_ENTRY;
      let queryHistories: QueryHistory[];
      if (pageToken) {
        // Dedupe by `name` on the append path. The backend uses
        // offset-based page tokens, so when `mergeLatest` prepends a
        // freshly-executed query the cached cursor is now off by one
        // (the new entry shifted every server position). The next
        // "Load more" call may return rows that already live in the
        // cache; filtering by `name` prevents visible duplicates.
        const cachedNames = new Set(prev.queryHistories.map((h) => h.name));
        const fresh = resp.queryHistories.filter(
          (h) => !cachedNames.has(h.name)
        );
        queryHistories = [...prev.queryHistories, ...fresh];
      } else {
        queryHistories = resp.queryHistories;
      }
      return {
        queryHistoryByKey: {
          ...s.queryHistoryByKey,
          [key]: { nextPageToken: resp.nextPageToken, queryHistories },
        },
      };
    });
    return resp;
  },

  resetPageToken: (filter) => {
    const key = getQueryHistoryCacheKey(filter);
    set((s) => ({
      queryHistoryByKey: {
        ...s.queryHistoryByKey,
        [key]: { queryHistories: [] },
      },
    }));
  },

  /**
   * Post-execute refresh that preserves pagination state. Fetches
   * page 1 (no cursor) and prepends entries whose `name` isn't already
   * cached. Keeps the existing `nextPageToken`, so a user who has
   * loaded pages 1–3 stays at pages 1–3 after running a new query —
   * the just-executed statement appears at the top without resetting
   * the list.
   */
  mergeLatest: async (filter) => {
    const key = getQueryHistoryCacheKey(filter);
    const resp = await sqlServiceClientConnect.searchQueryHistories(
      create(SearchQueryHistoriesRequestSchema, {
        pageSize: 5,
        filter: getListQueryHistoryFilter(filter),
      })
    );
    set((s) => {
      const existing = s.queryHistoryByKey[key];
      const existingNames = new Set(
        (existing?.queryHistories ?? []).map((h) => h.name)
      );
      const fresh = resp.queryHistories.filter(
        (h) => !existingNames.has(h.name)
      );
      return {
        queryHistoryByKey: {
          ...s.queryHistoryByKey,
          [key]: {
            nextPageToken: existing?.nextPageToken ?? resp.nextPageToken,
            queryHistories: [...fresh, ...(existing?.queryHistories ?? [])],
          },
        },
      };
    });
    return resp;
  },
});

/**
 * Curried selector for the query-history entry under a specific filter.
 * Returns a stable `EMPTY_ENTRY` reference when no entry exists yet, so
 * subscribers don't see infinite re-renders from default-object identity
 * changes.
 */
export const selectQueryHistoryEntry =
  (filter: QueryHistoryFilter) =>
  (state: SQLEditorStoreState): QueryHistoryEntry => {
    const key = getQueryHistoryCacheKey(filter);
    return state.queryHistoryByKey[key] ?? EMPTY_ENTRY;
  };
