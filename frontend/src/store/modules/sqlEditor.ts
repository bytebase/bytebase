import { SqlEditorState, ConnectionAtom } from "../../types";
import * as types from "../mutation-types";
import { makeActions } from "../actions";

const state: () => SqlEditorState = () => ({
  connectionTree: [],
  currentInstanceId: 6100,
  currentDatabaseId: 0,
  currentTableId: 0,
  queryStatement: "",
  queryResult: [],
});

const getters = {
  connectionTreeByInstanceId(state: SqlEditorState) {
    return state.connectionTree.find((item) => {
      return item.id === state.currentInstanceId;
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
    return `${state.currentInstanceId}/${state.currentDatabaseId}/${state.currentTableId}`;
  },
};

const actions = {
  ...makeActions({
    setSqlEditorState: types.SET_SQL_EDITOR_STATE,
    setConnectionTree: types.SET_CONNECTION_TREE,
    setQueryResult: types.SET_QUERY_RESULT,
  }),
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
