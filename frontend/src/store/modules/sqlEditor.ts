import { defineStore } from "pinia";
import {
  SQLEditorState,
  QueryInfo,
  QueryHistory,
  ActivitySQLEditorQueryPayload,
  SingleSQLResult,
} from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useLegacySQLStore } from "./sql";
import { useTabStore } from "./tab";
import { useDatabaseV1Store } from "./v1/database";
import { useInstanceV1Store, useSQLStore, useActivityV1Store } from "./v1";
import { QueryResult } from "@/types/proto/v1/sql_service";

// set the limit to 10000 temporarily to avoid the query timeout and page crash
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
    async executeQuery({ statement }: Pick<QueryInfo, "statement">) {
      const { instanceId, databaseId } = useTabStore().currentTab.connection;
      const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
      const databaseName =
        database.uid === String(UNKNOWN_ID) ? "" : database.databaseName;
      const instance = useInstanceV1Store().getInstanceByUID(instanceId);
      const response = await useSQLStore().queryReadonly({
        name: instance.name,
        connectionDatabase: databaseName,
        statement,
        limit: RESULT_ROWS_LIMIT,
      });

      return response;
    },
    async executeAdminQuery({ statement }: Pick<QueryInfo, "statement">) {
      const { instanceId, databaseId } = useTabStore().currentTab.connection;
      const database = useDatabaseV1Store().getDatabaseByUID(databaseId);
      const databaseName =
        database.uid === String(UNKNOWN_ID) ? "" : database.databaseName;
      const queryResult = await useLegacySQLStore().adminQuery({
        instanceId: Number(instanceId),
        databaseName,
        statement: statement,
        limit: RESULT_ROWS_LIMIT,
      });

      return queryResult;
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

export const mockAffectedRows0 = (): SingleSQLResult => {
  return {
    data: [["Affected Rows"], ["BIGINT"], [[0]], [false]],
    error: "",
  };
};

export const mockAffectedV1Rows0 = (): QueryResult => {
  return {
    columnNames: ["Affected Rows"],
    columnTypeNames: ["BIGINT"],
    masked: [false],
    error: "",
    statement: "",
    rows: [
      {
        values: [
          {
            int64Value: 0,
          },
        ],
      },
    ],
  };
};
