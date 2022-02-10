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
import { isDBAOrOwner, isDeveloper } from "../../utils";

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
    sharedSheet: null,
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
  isCreator: (
    state: SheetState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => {
    const currentUser = rootGetters["auth/currentUser"]();
    const currentSheet = getters.currentSheet;

    if (!currentSheet) return false;

    return currentUser.id === currentSheet!.creator.id;
  },
  accessIsPrivate: (state: SheetState, getters: any) => {
    return getters?.currentSheet?.visibility === "PRIVATE" ?? false;
  },
  accessIsPobject: (state: SheetState, getters: any) => {
    return getters?.currentSheet?.visibility === "PROJECT" ?? false;
  },
  accessIsPublic: (state: SheetState, getters: any) => {
    return getters?.currentSheet?.visibility === "PUBLIC" ?? false;
  },
  isReadOnly: (
    state: SheetState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => {
    const currentUser = rootGetters["auth/currentUser"]();
    const { currentSheet, accessIsPrivate, accessIsPobject, accessIsPublic } =
      getters;

    if (!currentSheet) return true;
    // creator/owner always can edit
    if (getters.isCreator) return false;

    // if current user is not owner, check the link access level
    return (
      accessIsPrivate ||
      (accessIsPobject && currentUser.role === "DEVELOPER") ||
      accessIsPublic
    );
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

    const newSheetList = state.sharedSheet
      ? sheetList.concat(state.sharedSheet)
      : sheetList;

    commit(
      types.SET_SHEET_LIST,
      newSheetList.sort((a, b) => b.createdTs - a.createdTs)
    );

    dispatch("setSheetState", { isFetchingSheet: false });
  },
  async fetchSheetById({ commit, dispatch, state }: any, id: number) {
    dispatch("setSheetState", { isFetchingSheet: true });

    const data = (await axios.get(`/api/sheet/${id}`)).data;
    const sheet = convertSheet(data.data, data.included);
    commit(types.SET_SHEET_BY_ID, sheet);

    // shared from others
    commit(types.SET_SHEET_STATE, { sharedSheet: sheet });

    dispatch("setSheetState", { isFetchingSheet: false });

    return sheet;
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
