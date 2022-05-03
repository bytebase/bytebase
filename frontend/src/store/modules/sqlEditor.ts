import { defineStore } from "pinia";
import dayjs from "dayjs";
import { isEmpty } from "lodash-es";
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

export const getDefaultConnectionContext = () => ({
  hasSlug: false,
  projectId: DEFAULT_PROJECT_ID,
  projectName: "",
  instanceId: UNKNOWN_ID,
  instanceName: "",
  databaseId: UNKNOWN_ID,
  databaseName: "",
  databaseType: "",
  tableId: UNKNOWN_ID,
  tableName: "",
  isLoadingTree: false,
  option: {},
});

export const useSQLEditorStore = defineStore("sqlEditor", {
  state: (): SQLEditorState => ({
    connectionTree: [],
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
      const ctx = state.connectionContext;
      return (
        ctx.instanceId === UNKNOWN_ID ||
        (ctx.databaseId === UNKNOWN_ID &&
          ctx.databaseType !== "MYSQL" &&
          ctx.databaseType !== "TIDB")
      );
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
      const queryResult = await useSQLStore().query({
        instanceId: this.connectionContext.instanceId,
        databaseName: this.connectionContext.databaseName,
        statement,
      });

      return queryResult;
    },
    async fetchConnectionByInstanceIdAndDatabaseId({
      instanceId,
      databaseId,
    }: Pick<SQLEditorState["connectionContext"], "instanceId" | "databaseId">) {
      const instance = await useInstanceStore().fetchInstanceById(instanceId);
      const database = await useDatabaseStore().fetchDatabaseById(databaseId);

      this.setConnectionContext({
        hasSlug: true,
        instanceId,
        instanceName: instance.name,
        databaseId,
        databaseName: database.name,
        databaseType: instance.engine,
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
  },
});
