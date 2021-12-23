import axios from "axios";
import { SqlEditorState, ConnectionAtom } from "../../types";

const state: () => SqlEditorState = () => ({
  connectionTree: [],
  currentInstanceId: 6001,
  currentDatabaseId: 0,
  currentTableId: 0,
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
      .map((item: ConnectionAtom) => rootGetters["table/tableListByDatabaseId"](item.id))
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
  setConnectionTree({ commit }: any, playload: ConnectionAtom[]) {
    commit("setConnectionTree", playload);
  },
};

const mutations = {
  setConnectionTree(state: SqlEditorState, payload: ConnectionAtom[]) {
    state.connectionTree = payload;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
