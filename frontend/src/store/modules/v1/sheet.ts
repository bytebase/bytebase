import { defineStore } from "pinia";
import { sheetServiceClient } from "@/grpcweb";
import { isEqual, isUndefined, isEmpty } from "lodash-es";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { useTabStore } from "../tab";
import { SheetId, UNKNOWN_ID } from "@/types";
import { useCurrentUserV1 } from "../auth";
import {
  getUserEmailFromIdentifier,
  sheetNamePrefix,
  projectNamePrefix,
  getProjectAndSheetId,
} from "./common";
import { isSheetReadableV1 } from "@/utils";

interface SheetState {
  sheetByName: Map<string, Sheet>;
}

export const useSheetV1Store = defineStore("sheet_v1", {
  state: (): SheetState => ({
    sheetByName: new Map<string, Sheet>(),
  }),

  getters: {
    currentSheet(state) {
      const currentTab = useTabStore().currentTab;

      if (!currentTab || isEmpty(currentTab)) {
        return;
      }

      // TODO: use resource id instead
      const sheetId = currentTab.sheetId || UNKNOWN_ID;
      for (const [_, sheet] of state.sheetByName) {
        const [_, uid] = getProjectAndSheetId(sheet.name);
        if (uid == sheetId) {
          return sheet;
        }
      }
    },
    isCreator() {
      const currentUserV1 = useCurrentUserV1();
      const currentSheet = this.currentSheet as Sheet;

      if (!currentSheet) return false;

      return (
        getUserEmailFromIdentifier(currentSheet.creator) ===
        currentUserV1.value.email
      );
    },
    /**
     * Check the sheet whether is read-only.
     * 1. If the sheet is not created yet, it cannot be edited.
     * 2. If the sheet is created by the current user, it can be edited.
     * 3. If the sheet is created by other user, will be checked the visibility of the sheet.
     *   a) If the sheet's visibility is private or public, it can be edited only if the current user is the creator of the sheet.
     *   b) If the sheet's visibility is project, will be checked whether the current user is the `OWNER` of the project, only the current user is the `OWNER` of the project, it can be edited.
     */
    isReadOnly() {
      const currentSheet = this.currentSheet as Sheet;

      // We don't have a selected sheet, we've got nothing to edit.
      if (!currentSheet) {
        return false;
      }

      // Incomplete sheets should be read-only. e.g. 100MB sheet from issue task.
      if (currentSheet.content.length !== currentSheet.contentSize) {
        return true;
      }

      return !isSheetReadableV1(currentSheet);
    },
  },

  actions: {
    setSheetList(sheets: Sheet[]) {
      for (const sheet of sheets) {
        this.sheetByName.set(sheet.name, sheet);
      }
    },
    async createSheet(parentPath: string, sheet: Partial<Sheet>) {
      const createdSheet = await sheetServiceClient.createSheet({
        parent: parentPath,
        sheet,
      });
      this.sheetByName.set(createdSheet.name, createdSheet);
      return createdSheet;
    },
    async upsertSheet(parentPath: string, sheet: Partial<Sheet>) {
      if (sheet.name) {
        const exist = this.sheetByName.get(sheet.name);
        if (!exist) {
          return;
        }
        return this.patchSheetById(getUpdateMaskForSheet(exist, sheet), sheet);
      }

      return this.createSheet(parentPath, sheet);
    },
    async patchSheetById(updateMask: string[], sheet: Partial<Sheet>) {
      const updatedSheet = await sheetServiceClient.updateSheet({
        sheet,
        updateMask,
      });
      this.sheetByName.set(updatedSheet.name, updatedSheet);
      return updatedSheet;
    },
    async fetchSheetById({
      parentPath,
      sheetId,
    }: {
      parentPath: string;
      sheetId: SheetId;
    }) {
      const sheet = await sheetServiceClient.getSheet({
        name: `${parentPath}/${sheetNamePrefix}${sheetId}`,
      });
      this.sheetByName.set(sheet.name, sheet);
      return sheet;
    },
    async getOrFetchSheetById({
      parentPath,
      sheetId,
    }: {
      parentPath: string;
      sheetId: SheetId;
    }) {
      const name = `${parentPath}/${sheetNamePrefix}${sheetId}`;
      const storedSheet = this.sheetByName.get(name);
      if (storedSheet) {
        return storedSheet;
      }
      return this.fetchSheetById({
        parentPath,
        sheetId,
      });
    },
    async fetchSharedSheetList() {
      const currentUserV1 = useCurrentUserV1();
      const { sheets } = await sheetServiceClient.searchSheets({
        parent: `${projectNamePrefix}-`,
        filter: `creator != users/${currentUserV1.value.email}`,
      });
      this.setSheetList(sheets);
      return sheets;
    },
    async fetchStarredSheetList() {
      const { sheets } = await sheetServiceClient.searchSheets({
        parent: `${projectNamePrefix}-`,
        filter: "starred = true",
      });
      this.setSheetList(sheets);
      return sheets;
    },
    async fetchMySheetList() {
      const currentUserV1 = useCurrentUserV1();
      const { sheets } = await sheetServiceClient.searchSheets({
        parent: `${projectNamePrefix}-`,
        filter: `creator = users/${currentUserV1.value.email}`,
      });
      this.setSheetList(sheets);
      return sheets;
    },
    async deleteSheetByName(name: string) {
      await sheetServiceClient.deleteSheet({ name });
      this.sheetByName.delete(name);
    },
    async syncSheetFromVCS(project: string) {
      await sheetServiceClient.syncSheets({
        parent: project,
      });
    },
  },
});

const getUpdateMaskForSheet = (
  origin: Sheet,
  update: Partial<Sheet>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (
    !isUndefined(update.content) &&
    !isEqual(origin.content, update.content)
  ) {
    updateMask.push("content");
  }
  if (
    !isUndefined(update.visibility) &&
    !isEqual(origin.visibility, update.visibility)
  ) {
    updateMask.push("visibility");
  }
  if (
    !isUndefined(update.starred) &&
    !isEqual(origin.starred, update.starred)
  ) {
    updateMask.push("starred");
  }
  if (
    !isUndefined(update.payload) &&
    !isEqual(origin.payload, update.payload)
  ) {
    updateMask.push("payload");
  }
  return updateMask;
};
