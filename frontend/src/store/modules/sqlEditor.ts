import { defineStore } from "pinia";
import {
  SQLEditorState,
  QueryInfo,
  QueryHistory,
  ActivitySQLEditorQueryPayload,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useInstanceV1Store, useSQLStore, useActivityV1Store } from "./v1";

// set the limit to 1000 temporarily to avoid the query timeout and page crash
export const RESULT_ROWS_LIMIT = 1000;

export const useSQLEditorStore = defineStore("sqlEditor", {
  state: (): SQLEditorState => ({
    isShowExecutingHint: false,
    shouldFormatContent: false,
    // Related data and status
    queryHistoryList: [],
    isFetchingQueryHistory: false,
    isFetchingSheet: false,
  }),

  actions: {
    setSQLEditorState(payload: Partial<SQLEditorState>) {
      Object.assign(this, payload);
    },
    setShouldFormatContent(payload: boolean) {
      this.shouldFormatContent = payload;
    },
    setQueryHistoryList(payload: QueryHistory[]) {
      this.queryHistoryList = payload;
    },
    setIsFetchingQueryHistory(payload: boolean) {
      this.isFetchingQueryHistory = payload;
    },
    async executeQuery(
      {
        instanceId,
        databaseName,
        dataSourceId,
        statement,
      }: Pick<
        QueryInfo,
        "instanceId" | "databaseName" | "dataSourceId" | "statement"
      >,
      signal: AbortSignal
    ) {
      const instance = useInstanceV1Store().getInstanceByUID(
        String(instanceId)
      );
      const response = await useSQLStore().queryReadonly(
        {
          name: instance.name,
          connectionDatabase: databaseName || "",
          dataSourceId: dataSourceId || "",
          statement,
          limit: RESULT_ROWS_LIMIT,
        },
        signal
      );

      return response;
    },
    async fetchQueryHistoryList() {
      this.setIsFetchingQueryHistory(true);
      const activityList =
        await useActivityV1Store().fetchActivityListForQueryHistory({
          limit: 20,
          order: "desc",
        });
      const queryHistoryList: QueryHistory[] = activityList.map(
        (history): QueryHistory => {
          const payload = JSON.parse(
            history.payload
          ) as ActivitySQLEditorQueryPayload;
          return {
            name: history.name,
            creator: history.creator,
            createTime: history.createTime ?? new Date(),
            statement: payload.statement,
            durationNs: payload.durationNs,
            instanceId: payload.instanceId || UNKNOWN_ID,
            instanceName: payload.instanceName,
            databaseId: payload.databaseId || UNKNOWN_ID,
            databaseName: payload.databaseName,
            error: payload.error,
          };
        }
      );
      this.setQueryHistoryList(queryHistoryList);
      this.setIsFetchingQueryHistory(false);
    },
  },
});
