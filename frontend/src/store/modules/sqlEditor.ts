import { isEmpty } from "lodash-es";

import {
  SqlEditorState,
  ConnectionAtom,
  QueryInfo,
  ConnectionContext,
} from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

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
  queryStatement: "",
  selectedStatement: "",
  queryResult: [],
});

const getters = {
  connectionTreeByInstanceId(state: SqlEditorState) {
    return state.connectionTree.find((item) => {
      return item.id === state.connectionContext.instanceId;
    });
  },
  connectionInfo(
    state: SqlEditorState,
    getter: any,
    rootState: any,
    rootGetters: any
  ) {
    return {
      allInstances: rootState.instance.instanceById,
      allDatabases: rootState.database.databaseListByInstanceId,
      allTables: rootState.table.tableListByDatabaseId,
    };
  },
  connectionInfoByInstanceId(
    state: SqlEditorState,
    getter: any,
    rootState: any,
    rootGetters: any
  ) {
    const instance = getter.connectionTreeByInstanceId;
    const databases = rootGetters["database/databaseListByInstanceId"](
      instance.id
    );

    const tables = instance.children
      .map((item: ConnectionAtom) =>
        rootGetters["table/tableListByDatabaseId"](item.id)
      )
      .flat();

    return {
      instance,
      databases,
      tables,
    };
  },
  currentSlug(state: SqlEditorState) {
    const connectionContext = state.connectionContext;
    return `${connectionContext.instanceId}/${connectionContext.databaseId}/${connectionContext.tableId}`;
  },
  isEmptyStatement(state: SqlEditorState) {
    return isEmpty(state.queryStatement);
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
  [types.SET_QUERY_RESULT](state: SqlEditorState, payload: Array<any>) {
    state.queryResult = payload;
  },
  [types.SET_CONNECTION_CONTEXT](
    state: SqlEditorState,
    payload: Partial<ConnectionContext>
  ) {
    Object.assign(state.connectionContext, payload);
  },
};

type SqlEditorActionsMap = {
  setSqlEditorState: typeof mutations.SET_SQL_EDITOR_STATE;
  setConnectionTree: typeof mutations.SET_CONNECTION_TREE;
  setQueryResult: typeof mutations.SET_QUERY_RESULT;
  setConnectionContext: typeof mutations.SET_CONNECTION_CONTEXT;
};

const actions = {
  ...makeActions<SqlEditorActionsMap>({
    setSqlEditorState: types.SET_SQL_EDITOR_STATE,
    setConnectionTree: types.SET_CONNECTION_TREE,
    setQueryResult: types.SET_QUERY_RESULT,
    setConnectionContext: types.SET_CONNECTION_CONTEXT,
  }),
  async executeQuery(
    { commit, dispatch, state }: any,
    payload: Partial<QueryInfo> = {}
  ) {
    const res = await dispatch(
      "sql/query",
      {
        instanceId: state.connectionContext.instanceId,
        databaseName: state.connectionContext.databaseName,
        statement: !isEmpty(state.selectedStatement)
          ? state.selectedStatement
          : state.queryStatement,
        ...payload,
      },
      { root: true }
    );
    commit(types.SET_QUERY_RESULT, res.data);
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
    commit(types.SET_SQL_EDITOR_STATE, {
      queryResult: [],
    });
    commit(types.SET_CONNECTION_CONTEXT, {
      hasSlug: true,
      instanceId,
      instanceName: instanceInfo.name,
      databaseId,
      databaseName: databaseInfo.name,
    });
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
