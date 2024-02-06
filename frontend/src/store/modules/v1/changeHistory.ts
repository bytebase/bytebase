import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
  GetChangeHistoryRequest,
  ListChangeHistoriesRequest,
} from "@/types/proto/v1/database_service";
import {
  extractChangeHistoryUID,
  extractDatabaseResourceName,
  getStatementSize,
} from "@/utils";

type CacheKeyType = [string /* name */, ChangeHistoryView];

export const useChangeHistoryStore = defineStore("changeHistory_v1", () => {
  const cache = useCache<CacheKeyType, ChangeHistory>(
    "bb.change-history.by-name"
  );

  // const changeHistoryMapByName = reactive(new Map<string, ChangeHistory>());
  const changeHistoryListMapByDatabase = reactive(
    new Map<string, ChangeHistory[]>()
  );

  // const upsertChangeHistoryMap = async (historyList: ChangeHistory[]) => {
  //   for (let i = 0; i < historyList.length; i++) {
  //     const history = historyList[i];
  //     changeHistoryMapByName.set(history.name, history);
  //   }
  // };
  const upsertChangeHistoryListMap = async (
    parent: string,
    historyList: ChangeHistory[]
  ) => {
    changeHistoryListMapByDatabase.set(parent, historyList);
    historyList.forEach((entity) => {
      cache.setEntity(
        [entity.name, ChangeHistoryView.CHANGE_HISTORY_VIEW_BASIC],
        entity
      );
    });
  };

  const fetchChangeHistoryList = async (
    params: Partial<ListChangeHistoriesRequest>
  ) => {
    const { parent } = params;
    if (!parent) throw new Error('"parent" field is required');
    const { changeHistories } = await databaseServiceClient.listChangeHistories(
      params
    );
    await upsertChangeHistoryListMap(parent, changeHistories);
    return changeHistories;
  };
  const getOrFetchChangeHistoryListOfDatabase = async (
    databaseName: string
  ) => {
    if (changeHistoryListMapByDatabase.has(databaseName)) {
      return changeHistoryListMapByDatabase.get(databaseName) ?? [];
    }
    return fetchChangeHistoryList({ parent: databaseName });
  };
  const changeHistoryListByDatabase = (name: string) => {
    return changeHistoryListMapByDatabase.get(name) ?? [];
  };
  const fetchChangeHistory = async (
    params: Partial<GetChangeHistoryRequest>
  ) => {
    const changeHistory = await databaseServiceClient.getChangeHistory(params);
    return changeHistory;
  };
  const getOrFetchChangeHistoryByName = async (
    name: string,
    view: ChangeHistoryView
  ) => {
    const uid = extractChangeHistoryUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const entity = cache.getEntity([name, view]);
    if (entity) {
      return entity;
    }
    const request = cache.getRequest([name, view]);
    if (request) {
      return request;
    }
    const promise = fetchChangeHistory({ name, view });
    cache.setRequest([name, view], promise);
    return promise;
  };
  /**
   * 
   * @param name 
   * @param view default undefined to any view (full -> basic)
   * @returns 
   */
  const getChangeHistoryByName = (
    name: string,
    view: ChangeHistoryView | undefined = undefined
  ) => {
    if (view === undefined) {
      return (
        cache.getEntity([name, ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL]) ??
        cache.getEntity([name, ChangeHistoryView.CHANGE_HISTORY_VIEW_BASIC])
      );
    }
    return cache.getEntity([name, view]);
  };
  const exportChangeHistoryFullStatementByName = async (
    name: string
  ): Promise<{
    changeHistory: ChangeHistory | undefined;
    type: "MIGRATE" | "BASELINE" | "UNSUPPORTED";
    statement: string;
  }> => {
    const changeHistory = await databaseServiceClient.getChangeHistory({
      name,
      view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
    });
    if (changeHistory) {
      if (
        changeHistory.type === ChangeHistory_Type.MIGRATE ||
        changeHistory.type === ChangeHistory_Type.MIGRATE_SDL ||
        changeHistory.type === ChangeHistory_Type.MIGRATE_GHOST ||
        changeHistory.type === ChangeHistory_Type.BRANCH ||
        changeHistory.type === ChangeHistory_Type.DATA
      ) {
        return {
          changeHistory,
          type: "MIGRATE",
          statement: changeHistory.statement,
        };
      } else if (changeHistory.type === ChangeHistory_Type.BASELINE) {
        return {
          changeHistory,
          type: "BASELINE",
          statement: changeHistory.schema,
        };
      }
    }
    return {
      changeHistory,
      type: "UNSUPPORTED",
      statement: "",
    };
  };

  return {
    fetchChangeHistoryList,
    getOrFetchChangeHistoryListOfDatabase,
    changeHistoryListByDatabase,
    fetchChangeHistory,
    getOrFetchChangeHistoryByName,
    getChangeHistoryByName,
    exportChangeHistoryFullStatementByName,
  };
});
