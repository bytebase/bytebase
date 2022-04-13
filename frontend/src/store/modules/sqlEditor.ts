import dayjs from "dayjs";
import { isEmpty } from "lodash-es";

import {
  SqlEditorState,
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
} from "@/types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";
import { unknown } from "@/types";
import {
  useActivityStore,
  useDatabaseStore,
  useInstanceStore,
  useTableStore,
  useSQLStore,
} from "../pinia-modules";

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

const state: () => SqlEditorState = () => ({
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
});

const getters = {
  connectionTreeByInstanceId(state: SqlEditorState): Partial<ConnectionAtom> {
    const idx = state.connectionTree.findIndex((item) => {
      return item.id === state.connectionContext.instanceId;
    });

    return idx !== -1 ? state.connectionTree[idx] : {};
  },
  connectionInfo(
    state: SqlEditorState,
    getter: any,
    rootState: any,
    rootGetters: any
  ) {
    return {
      projectListById: rootState.project.projectById,
      instanceListById: rootState.instance.instanceById,
      databaseListByInstanceId: rootState.database.databaseListByInstanceId,
      databaseListByProjectId: rootState.database.databaseListByProjectId,
      tableListByDatabaseId: rootState.table.tableListByDatabaseId,
    };
  },
  connectionInfoByInstanceId(
    state: SqlEditorState,
    getter: any,
    rootState: any,
    rootGetters: any
  ) {
    let instance = {} as any;
    let databaseList: Database[] = [];
    let tableList = [];

    if (!isEmpty(getter.connectionTreeByInstanceId)) {
      instance = getter.connectionTreeByInstanceId;
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
    (state: SqlEditorState, getter: any, rootState: any) =>
    (databaseId: DatabaseId): ProjectId => {
      let projectId = UNKNOWN_ID;
      const databaseListByProjectId =
        rootState.database.databaseListByProjectId;
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
  currentSlug(state: SqlEditorState) {
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
  isDisconnected(state: SqlEditorState) {
    const ctx = state.connectionContext;
    return (
      ctx.instanceId === UNKNOWN_ID ||
      (ctx.databaseId === UNKNOWN_ID &&
        ctx.databaseType !== "MYSQL" &&
        ctx.databaseType !== "TIDB")
    );
  },
};

const mutations = {
  [types.SET_SQL_EDITOR_STATE](
    state: SqlEditorState,
    payload: Partial<SqlEditorState>
  ) {
    Object.assign(state, payload);
  },
  [types.SET_CONNECTION_TREE](
    state: SqlEditorState,
    payload: ConnectionAtom[]
  ) {
    state.connectionTree = payload;
  },
  [types.SET_CONNECTION_CONTEXT](
    state: SqlEditorState,
    payload: Partial<ConnectionContext>
  ) {
    Object.assign(state.connectionContext, payload);
  },
  [types.SET_SHOULD_SET_CONTENT](state: SqlEditorState, payload: boolean) {
    state.shouldSetContent = payload;
  },
  [types.SET_SHOULD_FORMAT_CONTENT](state: SqlEditorState, payload: boolean) {
    state.shouldFormatContent = payload;
  },
  [types.SET_IS_EXECUTING](state: SqlEditorState, payload: boolean) {
    state.isExecuting = payload;
  },
  [types.SET_QUERY_HISTORY_LIST](
    state: SqlEditorState,
    payload: QueryHistory[]
  ) {
    state.queryHistoryList = payload;
  },
  [types.SET_IS_FETCHING_QUERY_HISTORY](
    state: SqlEditorState,
    payload: boolean
  ) {
    state.isFetchingQueryHistory = payload;
  },
};

type SqlEditorActionsMap = {
  setSqlEditorState: typeof mutations.SET_SQL_EDITOR_STATE;
  setConnectionTree: typeof mutations.SET_CONNECTION_TREE;
  setConnectionContext: typeof mutations.SET_CONNECTION_CONTEXT;
  setShouldSetContent: typeof mutations.SET_SHOULD_SET_CONTENT;
  setShouldFormatContent: typeof mutations.SET_SHOULD_FORMAT_CONTENT;
  setIsExecuting: typeof mutations.SET_IS_EXECUTING;
  setQueryHistoryList: typeof mutations.SET_QUERY_HISTORY_LIST;
  setIsFetchingQueryHistory: typeof mutations.SET_IS_FETCHING_QUERY_HISTORY;
};

const actions = {
  ...makeActions<SqlEditorActionsMap>({
    setSqlEditorState: types.SET_SQL_EDITOR_STATE,
    setConnectionTree: types.SET_CONNECTION_TREE,
    setConnectionContext: types.SET_CONNECTION_CONTEXT,
    setShouldSetContent: types.SET_SHOULD_SET_CONTENT,
    setShouldFormatContent: types.SET_SHOULD_FORMAT_CONTENT,
    setIsExecuting: types.SET_IS_EXECUTING,
    setQueryHistoryList: types.SET_QUERY_HISTORY_LIST,
    setIsFetchingQueryHistory: types.SET_IS_FETCHING_QUERY_HISTORY,
  }),
  async executeQuery(
    { dispatch, state }: any,
    { statement }: Pick<QueryInfo, "statement">
  ) {
    const queryResult = await useSQLStore().query({
      instanceId: state.connectionContext.instanceId,
      databaseName: state.connectionContext.databaseName,
      statement,
    });

    return queryResult;
  },
  async fetchConnectionByInstanceIdAndDatabaseId(
    { commit, dispatch }: any,
    {
      instanceId,
      databaseId,
    }: Pick<SqlEditorState["connectionContext"], "instanceId" | "databaseId">
  ) {
    const instance = await useInstanceStore().fetchInstanceById(instanceId);
    const database = await useDatabaseStore().fetchDatabaseById(databaseId);

    commit(types.SET_CONNECTION_CONTEXT, {
      hasSlug: true,
      instanceId,
      instanceName: instance.name,
      databaseId,
      databaseName: database.name,
      databaseType: instance.engine,
    });
  },
  async fetchQueryHistoryList({ commit, dispatch }: any) {
    commit(types.SET_IS_FETCHING_QUERY_HISTORY, true);
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

    commit(
      types.SET_QUERY_HISTORY_LIST,
      queryHistoryList.sort((a, b) => b.createdTs - a.createdTs)
    );
    commit(types.SET_IS_FETCHING_QUERY_HISTORY, false);
  },
  async deleteQueryHistory({ commit, dispatch, state }: any, id: number) {
    await useActivityStore().deleteActivityById(id);

    commit(
      types.SET_QUERY_HISTORY_LIST,
      state.queryHistoryList.filter((t: QueryHistory) => t.id !== id)
    );
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
