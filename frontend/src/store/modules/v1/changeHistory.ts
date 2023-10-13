import { defineStore } from "pinia";
import { reactive } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import {
  ChangeHistory,
  ChangeHistoryView,
  ChangeHistory_Type,
  GetChangeHistoryRequest,
  ListChangeHistoriesRequest,
} from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "@/utils";

export const useChangeHistoryStore = defineStore("changeHistory_v1", () => {
  const changeHistoryMapByName = reactive(new Map<string, ChangeHistory>());
  const changeHistoryListMapByDatabase = reactive(
    new Map<string, ChangeHistory[]>()
  );

  const upsertChangeHistoryMap = async (historyList: ChangeHistory[]) => {
    for (let i = 0; i < historyList.length; i++) {
      const history = historyList[i];
      changeHistoryMapByName.set(history.name, history);
    }
  };
  const upsertChangeHistoryListMap = async (
    parent: string,
    historyList: ChangeHistory[]
  ) => {
    changeHistoryListMapByDatabase.set(parent, historyList);
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
    await upsertChangeHistoryMap([changeHistory]);
    return changeHistory;
  };
  const getOrFetchChangeHistoryByName = async (name: string) => {
    const changeHistory = changeHistoryMapByName.get(name);
    if (changeHistory) return changeHistory;
    return await fetchChangeHistory({ name });
  };
  const getChangeHistoryByName = (name: string) => {
    const detail = changeHistoryMapByName.get(name);
    if (detail) {
      return detail;
    }
    const { full: parent } = extractDatabaseResourceName(name);
    const brief = changeHistoryListMapByDatabase
      .get(parent)
      ?.find((ch) => ch.name === name);
    return brief;
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
