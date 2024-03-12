import { defineStore } from "pinia";
import { ref } from "vue";
import { ActivitySQLEditorQueryPayload, SQLEditorQueryHistory } from "@/types";
import { useActivityV1Store } from "./v1";

export const useSQLEditorQueryHistoryStore = defineStore(
  "sqlEditorQueryHistory",
  () => {
    const isFetching = ref(false);
    const queryHistoryList = ref<SQLEditorQueryHistory[]>([]);

    const fetchQueryHistoryList = async () => {
      isFetching.value = true;
      const activityList =
        await useActivityV1Store().fetchActivityListForQueryHistory({
          limit: 20,
          order: "desc",
        });
      console.log(activityList);
      const historyList = activityList.map<SQLEditorQueryHistory>((history) => {
        const payload = JSON.parse(
          history.payload
        ) as ActivitySQLEditorQueryPayload;
        return {
          name: history.name,
          creator: history.creator,
          createTime: history.createTime ?? new Date(),
          statement: payload.statement,
          durationNs: payload.durationNs,
          instance: payload.instanceName,
          database: payload.databaseName,
          error: payload.error,
        };
      });
      isFetching.value = false;
      queryHistoryList.value = historyList;
    };

    return {
      isFetching,
      queryHistoryList,
      fetchQueryHistoryList,
    };
  }
);
