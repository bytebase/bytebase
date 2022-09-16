import { defineStore } from "pinia";
import dayjs from "dayjs";
import {
  SQLEditorState,
  ConnectionAtom,
  QueryInfo,
  ConnectionContext,
  Database,
  DatabaseId,
  ProjectId,
  QueryHistory,
  UNKNOWN_ID,
  Sheet,
  unknown,
  InstanceId,
  Connection,
} from "@/types";
import { useActivityStore } from "./activity";
import { useDatabaseStore } from "./database";
import { useInstanceStore } from "./instance";
import { useTableStore } from "./table";
import { useSQLStore } from "./sql";
import { useProjectStore } from "./project";
import { useTabStore } from "./tab";
import { emptyConnection } from "@/utils";

export const getDefaultConnectionContext = () => ({
  option: {} as any,
});

export const useSQLEditorStore = defineStore("sqlEditor", {
  state: (): SQLEditorState => ({
    connectionTree: [],
    connectionContext: getDefaultConnectionContext(),
    isLoadingTree: false,
    isExecuting: false,
    isShowExecutingHint: false,
    shouldFormatContent: false,
    // Related data and status
    queryHistoryList: [],
    isFetchingQueryHistory: false,
    isFetchingSheet: false,
    sharedSheet: unknown("SHEET") as Sheet,
  }),

  getters: {
    // TODO: remove this after a refactor to <TableSchema>
    connectionInfo() {
      const projectStore = useProjectStore();
      const instanceStore = useInstanceStore();
      const databaseStore = useDatabaseStore();
      const tableStore = useTableStore();

      return {
        projectListById: projectStore.projectById,
        instanceListById: instanceStore.instanceById,
        databaseListByInstanceId: databaseStore.databaseListByInstanceId,
        databaseListByProjectId: databaseStore.databaseListByProjectId,
        tableListByDatabaseId: tableStore.tableListByDatabaseId,
      };
    },
    findProjectIdByDatabaseId:
      (state) =>
      (databaseId: DatabaseId): ProjectId => {
        const databaseStore = useDatabaseStore();
        let projectId = UNKNOWN_ID;
        const databaseListByProjectId = databaseStore.databaseListByProjectId;
        for (const [id, databaseList] of databaseListByProjectId) {
          const idx = databaseList.findIndex(
            (database: Database) => database.id === databaseId
          );
          if (idx !== -1) {
            projectId = id;
            break;
          }
        }
        return projectId;
      },
  },

  actions: {
    setSQLEditorState(payload: Partial<SQLEditorState>) {
      Object.assign(this, payload);
    },
    setConnectionTree(payload: ConnectionAtom[]) {
      this.connectionTree = payload;
    },
    setConnectionContext(payload: Partial<ConnectionContext>) {
      Object.assign(this.connectionContext, payload);
    },
    setShouldFormatContent(payload: boolean) {
      this.shouldFormatContent = payload;
    },
    setIsExecuting(payload: boolean) {
      this.isExecuting = payload;
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
        // set the limit to 10000 temporarily to avoid the query timeout and page crash
        limit: 10000,
      });

      return queryResult;
    },
    async fetchConnectionByInstanceIdAndDatabaseId({
      instanceId,
      databaseId,
    }: {
      instanceId: InstanceId;
      databaseId: DatabaseId;
    }): Promise<Connection> {
      const [database] = await Promise.all([
        useDatabaseStore().getOrFetchDatabaseById(databaseId),
        useInstanceStore().getOrFetchInstanceById(instanceId),
        useTableStore().getOrFetchTableListByDatabaseId(databaseId),
      ]);

      return {
        ...emptyConnection(),
        projectId: database.project.id,
        instanceId,
        databaseId,
      };
    },
    async fetchQueryHistoryList() {
      this.setIsFetchingQueryHistory(true);
      const activityList =
        await useActivityStore().fetchActivityListForQueryHistory({
          limit: 50,
        });
      const queryHistoryList: QueryHistory[] = activityList.map(
        (history: any) => {
          return {
            id: history.id,
            creator: history.creator,
            createdTs: history.createdTs,
            updatedTs: history.updatedTs,
            statement: history.payload.statement,
            durationNs: history.payload.durationNs,
            instanceName: history.payload.instanceName,
            databaseName: history.payload.databaseName,
            error: history.payload.error,
            createdAt: dayjs(history.createdTs * 1000).format(
              "YYYY-MM-DD HH:mm:ss"
            ),
          };
        }
      );

      this.setQueryHistoryList(
        queryHistoryList.sort((a, b) => b.createdTs - a.createdTs)
      );
      this.setIsFetchingQueryHistory(false);
    },
    async deleteQueryHistory(id: number) {
      await useActivityStore().deleteActivityById(id);

      this.setQueryHistoryList(
        this.queryHistoryList.filter((t: QueryHistory) => t.id !== id)
      );
    },
  },
});
