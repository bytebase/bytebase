import { defineStore } from "pinia";
import dayjs from "dayjs";
import {
  SQLEditorState,
  QueryInfo,
  QueryHistory,
  ActivitySQLEditorQueryPayload,
  SingleSQLResult,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useActivityStore } from "./activity";
import { useDatabaseStore } from "./database";
import { useSQLStore } from "./sql";
import { useTabStore } from "./tab";

// set the limit to 10000 temporarily to avoid the query timeout and page crash
export const RESULT_ROWS_LIMIT = 10000;

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
    async executeQuery({ statement }: Pick<QueryInfo, "statement">) {
      const { instanceId, databaseId } = useTabStore().currentTab.connection;
      const database = useDatabaseStore().getDatabaseById(databaseId);
      const databaseName = database.id === UNKNOWN_ID ? "" : database.name;
      const queryResult = await useSQLStore().query({
        instanceId,
        databaseName,
        statement: statement,
        limit: RESULT_ROWS_LIMIT,
      });

      return queryResult;
    },
    async executeAdminQuery({ statement }: Pick<QueryInfo, "statement">) {
      const { instanceId, databaseId } = useTabStore().currentTab.connection;
      const database = useDatabaseStore().getDatabaseById(databaseId);
      const databaseName = database.id === UNKNOWN_ID ? "" : database.name;
      const queryResult = await useSQLStore().adminQuery({
        instanceId,
        databaseName,
        statement: statement,
        limit: RESULT_ROWS_LIMIT,
      });

      return queryResult;
    },
    async fetchQueryHistoryList() {
      this.setIsFetchingQueryHistory(true);
      const activityList =
        await useActivityStore().fetchActivityListForQueryHistory({
          limit: 20,
        });
      const queryHistoryList: QueryHistory[] = activityList.map((history) => {
        const payload = history.payload as ActivitySQLEditorQueryPayload;
        return {
          id: history.id,
          creator: history.creator,
          createdTs: history.createdTs,
          updatedTs: history.updatedTs,
          statement: payload.statement,
          durationNs: payload.durationNs,
          instanceId: payload.instanceId || UNKNOWN_ID,
          instanceName: payload.instanceName,
          databaseId: payload.databaseId || UNKNOWN_ID,
          databaseName: payload.databaseName,
          error: payload.error,
          createdAt: dayjs(history.createdTs * 1000).format(
            "YYYY-MM-DD HH:mm:ss"
          ),
        };
      });
      this.setQueryHistoryList(
        queryHistoryList.sort((a, b) => b.createdTs - a.createdTs)
      );
      this.setIsFetchingQueryHistory(false);
    },
  },
});

export const mockAffectedRows0 = (): SingleSQLResult => {
  return {
    data: [["Affected Rows"], ["BIGINT"], [[0]], [false]],
    error: "",
  };
};
