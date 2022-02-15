import axios from "axios";
import { isEmpty } from "lodash-es";

import * as types from "../mutation-types";
import { makeActions } from "../actions";
import type {
  Sheet,
  SheetId,
  SheetState,
  SheetPatch,
  CreateSheetState,
  Principal,
  ResourceObject,
  ConnectionContext,
  TabInfo,
} from "../../types";
import { empty, unknown } from "../../types";
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
  ): Sheet => {
    const currentTab = rootGetters["tab/currentTab"];

    if (!currentTab || isEmpty(currentTab)) return unknown("SHEET") as Sheet;

    const sheetId = currentTab.sheetId;

    return state.sheetById.get(sheetId) || (unknown("SHEET") as Sheet);
  },
};

const mutations = {
  [types.SET_SHEET_STATE](state: SheetState, payload: Partial<SheetState>) {
    Object.assign(state, payload);
  },
  [types.SET_SHEET_LIST](state: SheetState, payload: Sheet[]) {
    state.sheetList = payload;
  },
  [types.SET_SHEET_BY_ID](
    state: SheetState,
    { sheetId, sheet }: { sheetId: SheetId; sheet: Sheet }
  ) {
    state.sheetById.set(sheetId, sheet);
  },
  [types.DELETE_SHEET](state: SheetState, sheetId: SheetId) {
    const idx = state.sheetList.findIndex((sheet) => sheet.id === sheetId);
    if (idx !== -1) state.sheetList.splice(idx, 1);

    if (state.sheetById.has(sheetId)) {
      state.sheetById.delete(sheetId);
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
            name: tab.name,
            statement: tab.statement,
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

    commit(types.SET_SHEET_BY_ID, {
      sheetId: newSheet.id,
      sheet: newSheet,
    });

    return newSheet;
  },
  // retrieve
  async fetchSheetList({ commit, dispatch, state }: any) {
    dispatch("setSheetState", { isFetchingSheet: true });

    const data = (await axios.get(`/api/sheet`)).data;
    const sheetList: Sheet[] = data.data.map((sheet: ResourceObject) => {
      const newSheet = convertSheet(sheet, data.included);
      commit(types.SET_SHEET_BY_ID, {
        sheetId: newSheet.id,
        sheet: newSheet,
      });
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
    { id, name, statement, visibility }: SheetPatch
  ): Promise<Sheet> {
    const attributes: Omit<SheetPatch, "id"> = {};
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
    await axios.delete(`/api/sheet/${id}`);

    commit(types.DELETE_SHEET, id);
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
