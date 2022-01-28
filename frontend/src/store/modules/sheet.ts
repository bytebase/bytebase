import axios from "axios";

import * as types from "../mutation-types";
import { makeActions } from "../actions";
import type {
  Sheet,
  SheetState,
  Principal,
  ResourceIdentifier,
  ResourceObject,
} from "../../types";
import { getPrincipalFromIncludedList } from "./principal";

function convertSheet(
  sheet: ResourceObject,
  includedList: ResourceObject[]
): Sheet {
  const creatorId = (sheet.relationships!.creator.data as ResourceIdentifier)
    .id;
  const updaterId = (sheet.relationships!.updater.data as ResourceIdentifier)
    .id;

  return {
    ...(sheet.attributes as Omit<Sheet, "id" | "creator" | "updater">),
    creator: getPrincipalFromIncludedList(creatorId, includedList) as Principal,
    updater: getPrincipalFromIncludedList(updaterId, includedList) as Principal,
    id: parseInt(sheet.id),
  };
}

const state: () => SheetState = () => {
  return {
    sheetList: [],
  };
};

const getters = {};

const mutations = {
  [types.SET_SHEET_STATE](state: SheetState, payload: Partial<SheetState>) {
    Object.assign(state, payload);
  },
  [types.SET_SHEET_LIST](state: SheetState, payload: Sheet[]) {
    state.sheetList = payload;
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
  async createSheet(
    { commit, state }: any,
    {
      name,
      statement,
      visibility,
    }: { name: string; statement: string; visibility: string }
  ): Promise<Sheet> {
    const data = (
      await axios.post(`/api/sheet`, {
        data: {
          type: "createSheet",
          attributes: {
            name,
            statement,
            visibility,
          },
        },
      })
    ).data;
    const newSheet = convertSheet(data.data, data.included);

    commit(
      types.SET_SHEET_LIST,
      (state.sheetList as Sheet[])
        .concat(newSheet)
        .sort((a, b) => b.createdTs - a.createdTs)
    );

    return newSheet;
  },
  async fetchSheetList({ commit }: any) {
    commit(types.SET_IS_FETCHING_SAVED_QUERIES, true);
    const data = (await axios.get(`/api/sheet`)).data;
    const sheetList: Sheet[] = data.data.map((savedQuery: ResourceObject) => {
      return convertSheet(savedQuery, data.included);
    });

    commit(
      types.SET_SHEET_LIST,
      sheetList.sort((a, b) => b.createdTs - a.createdTs)
    );
    commit(types.SET_IS_FETCHING_SAVED_QUERIES, false);
  },
  async patchSheet(
    { dispatch }: any,
    {
      id,
      name,
      statement,
    }: {
      id: number;
      name?: string;
      statement?: string;
    }
  ) {
    const attributes: any = {};
    if (name) {
      attributes.name = name;
    }
    if (statement) {
      attributes.statement = statement;
    }

    await axios.patch(`/api/sheet/${id}`, {
      data: {
        type: "patchSheet",
        attributes,
      },
    });
    dispatch("fetchSheetList");
  },
  async deleteSheet({ commit, state }: any, id: number) {
    await axios.delete(`/api/sheet/${id}`);
    commit(
      types.SET_SHEET_LIST,
      state.sheetList.filter((t: Sheet) => t.id !== id)
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
