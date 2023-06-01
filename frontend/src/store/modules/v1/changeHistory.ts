import { defineStore } from "pinia";
import { reactive } from "vue";

import { databaseServiceClient } from "@/grpcweb";
import {
  ChangeHistory,
  GetChangeHistoryRequest,
  ListChangeHistoriesRequest,
} from "@/types/proto/v1/database_service";

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
    await upsertChangeHistoryMap(historyList);
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
  const getChangeHistoryByName = (name: string) => {
    return changeHistoryMapByName.get(name);
  };

  return {
    fetchChangeHistoryList,
    changeHistoryListByDatabase,
    fetchChangeHistory,
    getChangeHistoryByName,
  };
});
