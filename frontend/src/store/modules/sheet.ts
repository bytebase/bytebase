import axios from "axios";
import { isEmpty } from "lodash-es";

import * as types from "../mutation-types";
import { makeActions } from "../actions";
import type {
  Sheet,
  SheetId,
  SheetState,
  SheetPatch,
  Principal,
  ResourceObject,
  ConnectionContext,
  TabInfo,
  Database,
  Project,
  ProjectMember,
} from "@/types";
import { unknown, UNKNOWN_ID } from "@/types";
import { getPrincipalFromIncludedList } from "../pinia";
import { useAuthStore } from "../pinia-modules";

function convertSheet(
  sheet: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Sheet {
  let project = unknown("PROJECT") as Project;
  let database = unknown("DATABASE") as Database;

  const projectId = sheet.attributes.projectId;
  const databaseId = sheet.attributes.databaseId || UNKNOWN_ID;

  for (const item of includedList || []) {
    if (item.type == "project" && Number(item.id) === Number(projectId)) {
      project = rootGetters["project/convert"](item, includedList);
    }
    if (item.type == "database" && Number(item.id) == Number(databaseId)) {
      database = rootGetters["database/convert"](item, includedList);
    }
  }

  return {
    ...(sheet.attributes as Omit<Sheet, "id" | "creator" | "updater">),
    id: parseInt(sheet.id),
    creator: getPrincipalFromIncludedList(
      sheet.relationships!.creator.data,
      includedList
    ) as Principal,
    updater: getPrincipalFromIncludedList(
      sheet.relationships!.updater.data,
      includedList
    ) as Principal,
    project,
    database,
  };
}

const state: () => SheetState = () => {
  return {
    sheetList: [],
    sheetById: new Map(),
  };
};

const getters = {
  currentSheet:
    (state: SheetState) =>
    (currentTab: TabInfo): Sheet => {
      if (!currentTab || isEmpty(currentTab)) return unknown("SHEET") as Sheet;

      const sheetId = currentTab.sheetId || UNKNOWN_ID;

      return state.sheetById.get(sheetId) || (unknown("SHEET") as Sheet);
    },
  isCreator:
    (state: SheetState, getters: any, rootState: any, rootGetters: any) =>
    (currentTab: TabInfo): boolean => {
      const { currentUser } = useAuthStore();
      const currentSheet = getters.currentSheet(currentTab);

      if (!currentSheet) return false;

      return currentUser.id === currentSheet!.creator.id;
    },
  /**
   * Check the sheet whether is read-only.
   * 1、If the sheet is not created yet, it can not be edited.
   * 2、If the sheet is created by the current user, it can be edited.
   * 3、If the sheet is created by other user, will be checked the visibility of the sheet.
   *   a) If the sheet's visibility is private or public, it can be edited only if the current user is the creator of the sheet.
   *   b) If the sheet's visibility is project, will be checked whether the current user is the `OWNER` of the project, only the current user is the `OWNER` of the project, it can be edited.
   */
  isReadOnly:
    (state: SheetState, getters: any, rootState: any, rootGetters: any) =>
    (currentTab: TabInfo): boolean => {
      const { currentUser } = useAuthStore();
      const sharedSheet = rootState.sqlEditor.sharedSheet;
      const currentSheet = getters.currentSheet(currentTab);
      const isSharedByOthers = sharedSheet.id !== UNKNOWN_ID;

      if (!currentSheet) return true;
      // normal sheet can be edit by anyone
      if (!isSharedByOthers) return false;

      // if the sheet is shared by others, will be checked the visibility of the sheet.
      // creator always can edit
      if (getters.isCreator(currentTab)) return false;
      const isPrivate = currentSheet?.visibility === "PRIVATE" ?? false;
      const isProject = currentSheet?.visibility === "PROJECT" ?? false;
      const isPublic = currentSheet?.visibility === "PUBLIC" ?? false;

      const isCurrentUserProjectOwner = () => {
        const projectMemberList = currentSheet?.project.memberList;

        if (projectMemberList && projectMemberList.length > 0) {
          const currentMemberByProjectMember = projectMemberList?.find(
            (member: ProjectMember) => {
              return member.principal.id === currentUser.id;
            }
          );

          return currentMemberByProjectMember.role !== "OWNER";
        }

        return false;
      };

      // if current user is not creator, check the link access level by project relationship
      return (
        isPrivate || isPublic || (isProject && isCurrentUserProjectOwner())
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
  [types.SET_SHEET_BY_ID](
    state: SheetState,
    { sheetId, sheet }: { sheetId: SheetId; sheet: Sheet }
  ) {
    const item = state.sheetList.find((sheet) => sheet.id === sheetId);
    if (item !== undefined) {
      Object.assign(item, sheet);
    }
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
    currentTab: TabInfo
  ): Promise<Sheet> {
    const ctx = rootState.sqlEditor.connectionContext as ConnectionContext;

    const result = (
      await axios.post(`/api/sheet`, {
        data: {
          type: "createSheet",
          attributes: {
            projectId: ctx.projectId,
            databaseId: ctx.databaseId,
            name: currentTab.name,
            statement: currentTab.statement,
            visibility: "PRIVATE",
          },
        },
      })
    ).data;
    const sheet = convertSheet(result.data, result.included, rootGetters);

    commit(
      types.SET_SHEET_LIST,
      (state.sheetList as Sheet[])
        .concat(sheet)
        .sort((a, b) => b.createdTs - a.createdTs)
    );

    commit(types.SET_SHEET_BY_ID, {
      sheetId: sheet.id,
      sheet: sheet,
    });

    return sheet;
  },
  // retrieve
  async fetchSheetList({ commit, dispatch, state, rootGetters }: any) {
    dispatch(
      "sqlEditor/setSqlEditorState",
      { isFetchingSheet: true },
      { root: true }
    );
    const data = (await axios.get(`/api/sheet`)).data;
    const sheetList: Sheet[] = data.data.map((rawData: ResourceObject) => {
      const sheet = convertSheet(rawData, data.included, rootGetters);
      commit(types.SET_SHEET_BY_ID, {
        sheetId: sheet.id,
        sheet: sheet,
      });
      return sheet;
    });

    commit(
      types.SET_SHEET_LIST,
      sheetList.sort((a, b) => b.createdTs - a.createdTs)
    );
    dispatch(
      "sqlEditor/setSqlEditorState",
      { isFetchingSheet: false },
      { root: true }
    );
  },
  async fetchSheetById(
    { commit, dispatch, state, rootGetters }: any,
    sheetId: SheetId
  ) {
    const data = (await axios.get(`/api/sheet/${sheetId}`)).data;
    const sheet = convertSheet(data.data, data.included, rootGetters);
    commit(types.SET_SHEET_BY_ID, {
      sheetId: sheet.id,
      sheet: sheet,
    });

    return sheet;
  },
  // update
  async patchSheetById(
    { commit, dispatch, rootGetters }: any,
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

    const sheet = convertSheet(result.data, result.included, rootGetters);

    commit(types.SET_SHEET_BY_ID, {
      sheetId: sheet.id,
      sheet: sheet,
    });

    return sheet;
  },
  // delete
  async deleteSheet({ commit, state }: any, id: number) {
    await axios.delete(`/api/sheet/${id}`);

    commit(types.DELETE_SHEET, id);
  },
  // upsert
  async upsertSheet(
    { commit, dispatch, state }: any,
    payload: { sheet: Partial<Sheet>; currentTab: TabInfo }
  ): Promise<Sheet> {
    const { sheet, currentTab } = payload;
    const hasSheet = state.sheetById.has(sheet.id);

    if (hasSheet) {
      return dispatch("patchSheetById", sheet);
    } else {
      return dispatch("createSheet", currentTab);
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
