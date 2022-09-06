import { defineStore } from "pinia";
import dayjs from "dayjs";
import { cloneDeep, isEmpty } from "lodash-es";
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
  DEFAULT_PROJECT_ID,
  unknown,
} from "@/types";
import { useActivityStore } from "./activity";
import { useDatabaseStore } from "./database";
import { useInstanceStore } from "./instance";
import { useTableStore } from "./table";
import { useSQLStore } from "./sql";
import { useProjectStore } from "./project";
import { useTabStore } from "./tab";

export const getDefaultConnectionContext = (): ConnectionContext => ({
  projectId: DEFAULT_PROJECT_ID,
  instanceId: UNKNOWN_ID,
  databaseId: UNKNOWN_ID,
  tableId: UNKNOWN_ID,
  tableName: "",
  isLoadingTree: false,
});

export const useSQLEditorStore = defineStore("sqlEditor", {
  state: (): SQLEditorState => ({
    connectionTree: [],
    expandedTreeKeys: [],
    connectionContext: getDefaultConnectionContext(),
    isExecuting: false,
    isShowExecutingHint: false,
    shouldSetContent: false,
    shouldFormatContent: false,
    // Related data and status
    queryHistoryList: [],
    isFetchingQueryHistory: false,
    isFetchingSheet: false,
    sharedSheet: unknown("SHEET") as Sheet,
  }),

  getters: {
    connectionTreeByInstanceId(state): Partial<ConnectionAtom> {
      const idx = state.connectionTree.findIndex((item) => {
        return item.id === state.connectionContext.instanceId;
      });

      return idx !== -1 ? state.connectionTree[idx] : {};
    },

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
    connectionInfoByInstanceId() {
      let instance = {} as any;
      let databaseList: Database[] = [];
      let tableList = [];

      if (!isEmpty(this.connectionTreeByInstanceId)) {
        instance = this.connectionTreeByInstanceId;
        databaseList = useDatabaseStore().getDatabaseListByInstanceId(
          instance.id
        );

        tableList = instance.children
          .map((item: ConnectionAtom) =>
            useTableStore().getTableListByDatabaseId(item.id)
          )
          .flat();
      }

      return {
        instance,
        databaseList,
        tableList,
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

    currentSlug(state) {
      const connectionContext = state.connectionContext;
      return `${connectionContext.instanceId}/${connectionContext.databaseId}/${connectionContext.tableId}`;
    },
    /**
     * check the connection whether disconnected
     * 1、If the context is not set the instanceId, return true
     * 2、If the context is set the instanceId, but not set the databaseId and databaseType is not MYSQL or TIDB, return true
     * @param state
     * @returns boolean
     */
    isDisconnected(state) {
      const { instanceId, databaseId } = state.connectionContext;
      if (instanceId === UNKNOWN_ID) {
        return true;
      }
      const instance = useInstanceStore().getInstanceById(instanceId);
      if (instance.engine === "MYSQL" || instance.engine === "TIDB") {
        // Connecting to instance directly.
        return false;
      }
      return databaseId === UNKNOWN_ID;
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

      // When the connection context changed, save a copy of it to current tab.
      useTabStore().updateCurrentTab({
        connectionContext: cloneDeep(this.connectionContext),
      });

      const { instanceId, databaseId } = this.connectionContext;
      const keys: string[] = [];
      if (instanceId !== UNKNOWN_ID) keys.push(`instance-${instanceId}`);
      if (databaseId !== UNKNOWN_ID) keys.push(`database-${databaseId}`);
      this.addExpandedTreeKeys(keys);
    },
    setShouldSetContent(payload: boolean) {
      this.shouldSetContent = payload;
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
      const { instanceId, databaseId } = this.connectionContext;
      const databaseName =
        databaseId === UNKNOWN_ID
          ? undefined
          : useDatabaseStore().getDatabaseById(databaseId).name;

      const queryResult = await useSQLStore().query({
        instanceId,
        databaseName,
        statement,
        // set the limit to 10000 temporarily to avoid the query timeout and page crash
        limit: 10000,
      });

      return queryResult;
    },
    async fetchConnectionByInstanceIdAndDatabaseId({
      instanceId,
      databaseId,
    }: Pick<SQLEditorState["connectionContext"], "instanceId" | "databaseId">) {
      // Don't re-fetch instance/database/tableList every time.
      // Use cached data if possible.
      await useInstanceStore().getOrFetchInstanceById(instanceId);
      const database = await useDatabaseStore().getOrFetchDatabaseById(
        databaseId
      );
      await useTableStore().getOrFetchTableListByDatabaseId(database.id);

      this.setConnectionContext({
        instanceId,
        databaseId: database.syncStatus === "OK" ? database.id : undefined,
      });
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
    addExpandedTreeKeys(keys: string[]) {
      const set = new Set(this.expandedTreeKeys);
      keys.forEach((key) => {
        if (!set.has(key)) {
          this.expandedTreeKeys.push(key);
        }
      });
    },
  },
});
