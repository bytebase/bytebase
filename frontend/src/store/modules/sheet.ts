import { defineStore } from "pinia";
import axios from "axios";
import { isEmpty } from "lodash-es";
import {
  Sheet,
  SheetId,
  SheetState,
  SheetPatch,
  Principal,
  ResourceObject,
  TabInfo,
  Database,
  Project,
  ProjectMember,
  unknown,
  UNKNOWN_ID,
  SheetFind,
  SheetCreate,
  SheetOrganizerUpsert,
  ProjectId,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useAuthStore } from "./auth";
import { useDatabaseStore } from "./database";
import { useProjectStore } from "./project";
import { useSQLEditorStore } from "./sqlEditor";
import { useTabStore } from "./tab";

function convertSheet(
  sheet: ResourceObject,
  includedList: ResourceObject[]
): Sheet {
  let project = unknown("PROJECT") as Project;
  let database = unknown("DATABASE") as Database;

  const projectId = sheet.attributes.projectId;
  const databaseId = sheet.attributes.databaseId || UNKNOWN_ID;

  const databaseStore = useDatabaseStore();
  const projectStore = useProjectStore();
  for (const item of includedList || []) {
    if (item.type == "project" && Number(item.id) === Number(projectId)) {
      project = projectStore.convert(item, includedList);
    }
    if (item.type == "database" && Number(item.id) == Number(databaseId)) {
      database = databaseStore.convert(item, includedList);
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

export const useSheetStore = defineStore("sheet", {
  state: (): SheetState => ({
    sheetList: [],
    sheetById: new Map(),
  }),

  getters: {
    currentSheet(state) {
      const currentTab = useTabStore().currentTab;

      if (!currentTab || isEmpty(currentTab)) {
        return unknown("SHEET");
      }

      const sheetId = currentTab.sheetId || UNKNOWN_ID;

      return state.sheetById.get(sheetId) || (unknown("SHEET") as Sheet);
    },
    isCreator() {
      const { currentUser } = useAuthStore();
      const currentSheet = this.currentSheet as Sheet;

      if (!currentSheet) return false;

      return currentUser.id === currentSheet!.creator.id;
    },
    /**
     * Check the sheet whether is read-only.
     * 1. If the sheet is not created yet, it can not be edited.
     * 2. If the sheet is created by the current user, it can be edited.
     * 3. If the sheet is created by other user, will be checked the visibility of the sheet.
     *   a) If the sheet's visibility is private or public, it can be edited only if the current user is the creator of the sheet.
     *   b) If the sheet's visibility is project, will be checked whether the current user is the `OWNER` of the project, only the current user is the `OWNER` of the project, it can be edited.
     */
    isReadOnly() {
      const sqlEdtiorStore = useSQLEditorStore();
      const { currentUser } = useAuthStore();
      const sharedSheet = sqlEdtiorStore.sharedSheet;
      const currentSheet = this.currentSheet as Sheet;
      const isSharedByOthers = sharedSheet.id !== UNKNOWN_ID;

      if (!currentSheet) return true;
      // normal sheet can be edit by anyone
      if (!isSharedByOthers) return false;

      // if the sheet is shared by others, will be checked the visibility of the sheet.
      // creator always can edit
      if (this.isCreator) return false;
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
          ) as ProjectMember;

          return currentMemberByProjectMember.role !== "OWNER";
        }

        return false;
      };

      // if current user is not creator, check the link access level by project relationship
      return (
        isPrivate || isPublic || (isProject && isCurrentUserProjectOwner())
      );
    },
  },

  actions: {
    setSheetList(payload: Sheet[]) {
      this.sheetList = payload;
    },
    setSheetById({ sheetId, sheet }: { sheetId: SheetId; sheet: Sheet }) {
      const item = this.sheetList.find((sheet) => sheet.id === sheetId);
      if (item !== undefined) {
        Object.assign(item, sheet);
      }
      this.sheetById.set(sheetId, sheet);
    },
    upsertSheet(payload: {
      sheet: Pick<Sheet, "id" | "name" | "statement">;
      currentTab: TabInfo;
    }): Promise<Sheet> {
      const { sheet, currentTab } = payload;
      const hasSheet = this.sheetById.has(sheet.id);

      if (hasSheet) {
        return this.patchSheetById(sheet);
      } else {
        return this.createSheetByTab(currentTab);
      }
    },
    async createSheetByTab(currentTab: TabInfo): Promise<Sheet> {
      const sqlEditorStore = useSQLEditorStore();
      const ctx = sqlEditorStore.connectionContext;

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
      const sheet = convertSheet(result.data, result.included);

      this.setSheetList(
        this.sheetList.concat(sheet).sort((a, b) => b.createdTs - a.createdTs)
      );

      this.setSheetById({
        sheetId: sheet.id,
        sheet: sheet,
      });

      return sheet;
    },
    async createSheet(sheetCreate: SheetCreate): Promise<Sheet> {
      const resData = (
        await axios.post(`/api/sheet`, {
          data: {
            type: "createSheet",
            attributes: sheetCreate,
          },
        })
      ).data;
      const sheet = convertSheet(resData.data, resData.included);

      this.setSheetList(
        this.sheetList.concat(sheet).sort((a, b) => b.createdTs - a.createdTs)
      );
      this.setSheetById({
        sheetId: sheet.id,
        sheet: sheet,
      });

      return sheet;
    },
    async fetchMySheetList(sheetFind?: SheetFind) {
      const queryList = [];
      if (sheetFind?.projectId) {
        queryList.push(`projectId=${sheetFind.projectId}`);
      }
      if (sheetFind?.databaseId) {
        queryList.push(`databaseId=${sheetFind.databaseId}`);
      }
      const resData = (await axios.get(`/api/sheet/my?${queryList.join("&")}`))
        .data;
      const sheetList: Sheet[] = resData.data.map((rawData: ResourceObject) => {
        const sheet = convertSheet(rawData, resData.included);
        this.setSheetById({
          sheetId: sheet.id,
          sheet: sheet,
        });
        return sheet;
      });

      sheetList.sort((a, b) => b.createdTs - a.createdTs);
      this.setSheetList(sheetList);

      return sheetList;
    },
    async fetchSharedSheetList(sheetFind?: SheetFind) {
      const queryList = [];
      if (sheetFind?.projectId) {
        queryList.push(`projectId=${sheetFind.projectId}`);
      }
      if (sheetFind?.databaseId) {
        queryList.push(`databaseId=${sheetFind.databaseId}`);
      }
      const resData = (
        await axios.get(`/api/sheet/shared?${queryList.join("&")}`)
      ).data;
      const sheetList: Sheet[] = resData.data.map((rawData: ResourceObject) => {
        const sheet = convertSheet(rawData, resData.included);
        this.setSheetById({
          sheetId: sheet.id,
          sheet: sheet,
        });
        return sheet;
      });

      sheetList.sort((a, b) => b.createdTs - a.createdTs);
      this.setSheetList(sheetList);

      return sheetList;
    },
    async fetchStarredSheetList(sheetFind?: SheetFind) {
      const queryList = [];
      if (sheetFind?.projectId) {
        queryList.push(`projectId=${sheetFind.projectId}`);
      }
      if (sheetFind?.databaseId) {
        queryList.push(`databaseId=${sheetFind.databaseId}`);
      }
      const resData = (
        await axios.get(`/api/sheet/starred?${queryList.join("&")}`)
      ).data;
      const sheetList: Sheet[] = resData.data.map((rawData: ResourceObject) => {
        const sheet = convertSheet(rawData, resData.included);
        this.setSheetById({
          sheetId: sheet.id,
          sheet: sheet,
        });
        return sheet;
      });

      sheetList.sort((a, b) => b.createdTs - a.createdTs);
      this.setSheetList(sheetList);

      return sheetList;
    },
    async fetchSheetById(sheetId: SheetId) {
      const data = (await axios.get(`/api/sheet/${sheetId}`)).data;
      const sheet = convertSheet(data.data, data.included);
      this.setSheetById({
        sheetId: sheet.id,
        sheet: sheet,
      });

      return sheet;
    },
    async patchSheetById(sheetPatch: SheetPatch): Promise<Sheet> {
      const resData = (
        await axios.patch(`/api/sheet/${sheetPatch.id}`, {
          data: {
            type: "sheetPatch",
            attributes: sheetPatch,
          },
        })
      ).data;

      const sheet = convertSheet(resData.data, resData.included);

      this.setSheetById({
        sheetId: sheet.id,
        sheet: sheet,
      });

      return sheet;
    },
    async upsertSheetOrganizer(sheetOrganizerUpsert: SheetOrganizerUpsert) {
      await axios.patch(`/api/sheet/${sheetOrganizerUpsert.sheeId}/organizer`, {
        data: {
          type: "sheetOrganizerUpsert",
          attributes: sheetOrganizerUpsert,
        },
      });
    },
    async deleteSheetById(sheetId: SheetId) {
      await axios.delete(`/api/sheet/${sheetId}`);

      const idx = this.sheetList.findIndex((sheet) => sheet.id === sheetId);
      if (idx !== -1) {
        this.sheetList.splice(idx, 1);
      }

      if (this.sheetById.has(sheetId)) {
        this.sheetById.delete(sheetId);
      }
    },
    async syncSheetFromVCS(projectId: ProjectId) {
      await axios.post(`/api/sheet/project/${projectId}/sync`);
    },
  },
});
