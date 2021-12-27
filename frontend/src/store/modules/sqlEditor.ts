import { isEmpty } from "lodash-es";

import { SqlEditorState, ConnectionAtom, QueryInfo } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

const state: () => SqlEditorState = () => ({
  connectionTree: [],
  connectionMeta: {
    instanceId: 6100,
    instanceName: "",
    databaseId: 0,
    databaseName: "",
    tableId: 0,
    tableName: "",
  },
  queryStatement: "",
  selectedStatement: "",
  queryResult: [],
});

const getters = {
  connectionTreeByInstanceId(state: SqlEditorState) {
    return state.connectionTree.find((item) => {
      return item.id === state.connectionMeta.instanceId;
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
    const connectionMeta = state.connectionMeta;
    return `${connectionMeta.instanceId}/${connectionMeta.databaseId}/${connectionMeta.tableId}`;
  },
  isEmptyStatement(state: SqlEditorState) {
    return isEmpty(state.queryStatement);
  },
};

const actions = {
  ...makeActions({
    setSqlEditorState: types.SET_SQL_EDITOR_STATE,
    setConnectionTree: types.SET_CONNECTION_TREE,
    setQueryResult: types.SET_QUERY_RESULT,
  }),
  async executeQueries(
    { commit, dispatch, state }: any,
    payload: Partial<QueryInfo>
  ) {
    const res = await dispatch(
      "sql/query",
      {
        instanceId: state.connectionMeta.instanceId,
        statement: !isEmpty(state.selectedStatement)
          ? state.selectedStatement
          : state.queryStatement,
        ...payload,
      },
      { root: true }
    );
    commit(types.SET_QUERY_RESULT, res.data);
    return res.data;
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
