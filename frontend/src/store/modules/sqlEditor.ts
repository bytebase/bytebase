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
} from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";
import dayjs from "dayjs";

const state: () => SqlEditorState = () => ({
  connectionTree: [],
  connectionContext: {
    hasSlug: false,
    instanceId: 0,
    instanceName: "",
    databaseId: 0,
    databaseName: "",
    tableId: 0,
    tableName: "",
    isLoadingTree: false,
    selectedDatabaseId: 0,
    selectedTableName: "",
  },
  shouldSetContent: false,
  queryHistoryList: [],
  isFetchingQueryHistory: false,
  isExecuting: false,
  isShowExecutingHint: false,
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
    let databaseList = [];
    let tableList = [];

    if (!isEmpty(getter.connectionTreeByInstanceId)) {
      instance = getter.connectionTreeByInstanceId;
      databaseList = rootGetters["database/databaseListByInstanceId"](
        instance.id
      );

      tableList = instance.children
        .map((item: ConnectionAtom) =>
          rootGetters["table/tableListByDatabaseId"](item.id)
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
    (state: SqlEditorState, getter: any) =>
    (databaseId: DatabaseId): ProjectId => {
      let projectId = 1;
      const databaseListByProjectId =
        getter.connectionInfo.databaseListByProjectId;
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
  [types.SET_IS_EXECUTING](state: SqlEditorState, payload: boolean) {
    state.isExecuting = payload;
  },
};

type SqlEditorActionsMap = {
  setSqlEditorState: typeof mutations.SET_SQL_EDITOR_STATE;
  setConnectionTree: typeof mutations.SET_CONNECTION_TREE;
  setConnectionContext: typeof mutations.SET_CONNECTION_CONTEXT;
  setShouldSetContent: typeof mutations.SET_SHOULD_SET_CONTENT;
  setQueryHistoryList: typeof mutations.SET_QUERY_HISTORY_LIST;
  setIsFetchingQueryHistory: typeof mutations.SET_IS_FETCHING_QUERY_HISTORY;
  setIsExecuting: typeof mutations.SET_IS_EXECUTING;
};

const actions = {
  ...makeActions<SqlEditorActionsMap>({
    setSqlEditorState: types.SET_SQL_EDITOR_STATE,
    setConnectionTree: types.SET_CONNECTION_TREE,
    setConnectionContext: types.SET_CONNECTION_CONTEXT,
    setShouldSetContent: types.SET_SHOULD_SET_CONTENT,
    setQueryHistory: types.SET_QUERY_HISTORY_LIST,
    setIsFetchingQueryHistory: types.SET_IS_FETCHING_QUERY_HISTORY,
    setIsExecuting: types.SET_IS_EXECUTING,
  }),
  async executeQuery(
    { dispatch, state, rootGetters }: any,
    payload: Partial<QueryInfo> = {}
  ) {
    const currentTab = rootGetters["editorSelector/currentTab"];
    const res = await dispatch(
      "sql/query",
      {
        instanceId: state.connectionContext.instanceId,
        databaseName: state.connectionContext.databaseName,
        statement: currentTab.selectedStatement || currentTab.queryStatement,
        ...payload,
      },
      { root: true }
    );

    dispatch(
      "editorSelector/updateActiveTab",
      {
        queryResult: res.data,
      },
      { root: true }
    );
    dispatch("sqlEditor/fetchQueryHistory", {}, { root: true });
    return res;
  },
  async fetchConnectionByInstanceIdAndDatabaseId(
    { commit, dispatch }: any,
    { instanceId, databaseId }: Partial<SqlEditorState["connectionContext"]>
  ) {
    const instanceInfo = await dispatch(
      "instance/fetchInstanceById",
      instanceId,
      { root: true }
    );
    const databaseInfo = await dispatch(
      "database/fetchDatabaseById",
      { databaseId },
      { root: true }
    );
    commit(types.SET_CONNECTION_CONTEXT, {
      hasSlug: true,
      instanceId,
      instanceName: instanceInfo.name,
      databaseId,
      databaseName: databaseInfo.name,
    });
  },
  async fetchQueryHistory({ commit, dispatch, rootGetters }: any) {
    commit(types.SET_IS_FETCHING_QUERY_HISTORY, true);
    const currentUser = rootGetters["auth/currentUser"]();
    const activityList = await dispatch(
      "activity/fetchActivityListForQueryHistory",
      currentUser.id,
      {
        root: true,
      }
    );
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
  async deleteQueryHistory({ dispatch }: any, id: number) {
    await dispatch("activity/deleteActivityById", id, {
      root: true,
    });
    dispatch("sqlEditor/fetchQueryHistory", {}, { root: true });
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
