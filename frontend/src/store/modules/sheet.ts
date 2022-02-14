import axios from "axios";
import { isEmpty } from "lodash-es";

import * as types from "../mutation-types";
import { makeActions } from "../actions";
import type {
  Sheet,
  SheetState,
  CreateSheetState,
  Principal,
  ResourceObject,
  ConnectionContext,
  TabInfo,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convertSheet(
  sheet: ResourceObject,
  includedList: ResourceObject[]
): Sheet {
  return {
    ...(sheet.attributes as Omit<Sheet, "id" | "creator" | "updater">),
    creator: getPrincipalFromIncludedList(
      sheet.relationships!.creator.data,
      includedList
    ) as Principal,
    updater: getPrincipalFromIncludedList(
      sheet.relationships!.updater.data,
      includedList
    ) as Principal,
    id: parseInt(sheet.id),
  };
}

const state: () => SheetState = () => {
  return {
    sheetList: [],
    sheetById: new Map(),
    isFetchingSheet: false,
  };
};

const getters = {
  currentSheet: (
    state: SheetState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => {
    const currentTab = rootGetters["tab/currentTab"];

    if (!currentTab || isEmpty(currentTab)) return null;

    const sheetId = currentTab.sheetId;

    return state.sheetById.get(sheetId);
  },
};

const mutations = {
  [types.SET_SHEET_STATE](state: SheetState, payload: Partial<SheetState>) {
    Object.assign(state, payload);
  },
  [types.SET_SHEET_LIST](state: SheetState, payload: Sheet[]) {
    state.sheetList = payload;
  },
  [types.SET_SHEET_BY_ID](state: SheetState, payload: Sheet) {
    state.sheetById.set(payload.id, payload);
  },
  [types.REMOVE_SHEET](state: SheetState, payload: Sheet) {
    state.sheetList.splice(state.sheetList.indexOf(payload), 1);

    if (state.sheetById.has(payload.id)) {
      state.sheetById.delete(payload.id);
    }
  },
};

type ActionsMap = {
  setSheetState: typeof mutations.SET_SHEET_STATE;
  setSheetList: typeof mutations.SET_SHEET_LIST;
};

const actions = {
  ...makeActions<ActionsMap>({
    setSheetState: types.SET_SHEET_STATE,
    setSheetList: types.SET_SHEET_LIST,
  }),
  // create
  async createSheet(
    { commit, state, rootState, rootGetters }: any,
    sheetRecord?: CreateSheetState
  ): Promise<Sheet> {
    const ctx = rootState.sqlEditor.connectionContext as ConnectionContext;
    const instance = rootGetters["instance/instanceById"](ctx.instanceId);
    const database = rootGetters["database/databaseById"](ctx.databaseId);
    const tab = rootGetters["tab/currentTab"] as TabInfo;

    const result = (
      await axios.post(`/api/sheet`, {
        data: {
          type: "createSheet",
          attributes: {
            instanceId: ctx.instanceId,
            databaseId: ctx.databaseId,
            name: tab.label,
            statement: tab.queryStatement,
            visibility: "PRIVATE",
          },
        },
      })
    ).data;
    const newSheet = convertSheet(result.data, result.included);
    newSheet.instance = instance;
    newSheet.database = database;

    commit(
      types.SET_SHEET_LIST,
      (state.sheetList as Sheet[])
        .concat(newSheet)
        .sort((a, b) => b.createdTs - a.createdTs)
    );

    commit(types.SET_SHEET_BY_ID, newSheet);

    return newSheet;
  },
  // retrieve
  async fetchSheetList({ commit, dispatch, state }: any) {
    dispatch("setSheetState", { isFetchingSheet: true });

    const data = (await axios.get(`/api/sheet`)).data;
    const sheetList: Sheet[] = data.data.map((sheet: ResourceObject) => {
      const newSheet = convertSheet(sheet, data.included);
      commit(types.SET_SHEET_BY_ID, newSheet);
      return newSheet;
    });

    commit(
      types.SET_SHEET_LIST,
      sheetList.sort((a, b) => b.createdTs - a.createdTs)
    );

    dispatch("setSheetState", { isFetchingSheet: false });
  },
  // update
  async patchSheetById(
    { dispatch }: any,
    { id, name, statement, visibility }: Partial<Sheet>
  ): Promise<Sheet> {
    const attributes: Partial<
      Pick<Sheet, "name" | "statement" | "visibility">
    > = {};
    if (name) {
      attributes.name = name;
    }
    if (statement) {
      attributes.statement = statement;
    }
    if (visibility) {
      attributes.visibility = visibility;
    }

    const result = (
      await axios.patch(`/api/sheet/${id}`, {
        data: {
          type: "sheetPatch",
          attributes,
        },
      })
    ).data;

    const newSheet = convertSheet(result.data, result.included);

    dispatch("fetchSheetList");

    return newSheet;
  },
  // delete
  async deleteSheet({ commit, state }: any, id: number) {
    const sheet = state.sheetById.get(id);

    await axios.delete(`/api/sheet/${id}`);
    commit(types.REMOVE_SHEET, sheet);
  },
  // upsert
  async upsertSheet(
    { commit, dispatch, state }: any,
    payload: Partial<Sheet>
  ): Promise<Sheet> {
    const hasSheet = state.sheetById.has(payload.id);

    if (hasSheet) {
      return dispatch("patchSheetById", payload);
    } else {
      return dispatch("createSheet");
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
