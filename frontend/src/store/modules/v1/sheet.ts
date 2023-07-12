import { defineStore } from "pinia";
import { computed, ref, unref, watch, watchEffect } from "vue";
import { sheetServiceClient } from "@/grpcweb";
import { isEqual, isUndefined, isEmpty } from "lodash-es";
import { Sheet, SheetOrganizer } from "@/types/proto/v1/sheet_service";
import { useTabStore } from "../tab";
import { useCurrentUserV1 } from "../auth";
import {
  getUserEmailFromIdentifier,
  projectNamePrefix,
  sheetNamePrefix,
  getProjectAndSheetId,
} from "./common";
import { extractSheetUID, isSheetReadableV1 } from "@/utils";
import { UNKNOWN_ID, SheetId, MaybeRef } from "@/types";

interface SheetState {
  sheetByName: Map<string, Sheet>;
}

const REQUEST_CACHE = new Map<
  string /* sheetResourceName */,
  Promise<Sheet | undefined>
>();

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

      const sheetName = currentTab.sheetName;
      if (!sheetName) {
        return;
      }
      return state.sheetByName.get(sheetName);
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
    getSheetUid(name: string): number {
      const [_, sheetId] = getProjectAndSheetId(name);
      if (!sheetId || Number.isNaN(sheetId)) {
        return UNKNOWN_ID;
      }
      return Number(sheetId);
    },
    getProjectResourceId(name: string): string {
      const [projectId, _] = getProjectAndSheetId(name);
      return projectId;
    },
    getSheetParentPath(name: string): string {
      const projectId = this.getProjectResourceId(name);
      return `${projectNamePrefix}${projectId}`;
    },
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
    async patchSheet(sheet: Partial<Sheet>) {
      if (!sheet.name) {
        return;
      }
      const exist = this.sheetByName.get(sheet.name);
      if (!exist) {
        return;
      }

      const updateMask = getUpdateMaskForSheet(exist, sheet);
      if (updateMask.length === 0) {
        return exist;
      }
      const updatedSheet = await this.patchSheetWithUpdateMask(
        updateMask,
        sheet
      );
      this.sheetByName.set(updatedSheet.name, updatedSheet);
      return updatedSheet;
    },
    async patchSheetWithUpdateMask(
      updateMask: string[],
      sheet: Partial<Sheet>
    ) {
      const updatedSheet = await sheetServiceClient.updateSheet({
        sheet,
        updateMask,
      });
      this.sheetByName.set(updatedSheet.name, updatedSheet);
      return updatedSheet;
    },
    async fetchSheetByName(name: string) {
      const cached = REQUEST_CACHE.get(name);
      if (cached) {
        return cached;
      }

      const runner = async () => {
        try {
          return await sheetServiceClient.getSheet({
            name,
          });
        } catch {
          return undefined;
        }
      };

      const request = runner();
      request.then((sheet) => {
        if (sheet) {
          this.sheetByName.set(sheet.name, sheet);
        } else {
          // If the request failed (e.g., "Too many requests")
          // Remove the cache entry so we can retry when needed.
          REQUEST_CACHE.delete(name);
        }
      });
      REQUEST_CACHE.set(name, request);
      return request;
    },
    getSheetByName(name: string) {
      const sheet = this.sheetByName.get(name);
      return sheet;
    },
    getSheetByUid(uid: SheetId) {
      for (const [name, sheet] of this.sheetByName) {
        if (`${this.getSheetUid(name)}` === `${uid}`) {
          return sheet;
        }
      }
    },
    async getOrFetchSheetByUid(uid: SheetId) {
      if (uid === undefined || uid === "undefined") {
        console.warn("undefined sheet uid");
        return;
      }
      if (uid === UNKNOWN_ID) {
        return;
      }
      const sheet = this.getSheetByUid(uid);
      if (sheet) {
        return sheet;
      }

      return this.fetchSheetByName(
        `${projectNamePrefix}-/${sheetNamePrefix}${uid}`
      );
    },
    async getOrFetchSheetByName(name: string) {
      const storedSheet = this.sheetByName.get(name);
      if (storedSheet) {
        return storedSheet;
      }
      return this.fetchSheetByName(name);
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
    async upsertSheetOrganizer(organizer: Partial<SheetOrganizer>) {
      await sheetServiceClient.updateSheetOrganizer({
        organizer,
        // for now we only support change the `starred` field.
        updateMask: ["starred"],
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
    !isUndefined(update.payload) &&
    !isEqual(origin.payload, update.payload)
  ) {
    updateMask.push("payload");
  }
  return updateMask;
};

export const useSheetStatementByUid = (sheetId: MaybeRef<SheetId>) => {
  const store = useSheetV1Store();
  watchEffect(async () => {
    await store.getOrFetchSheetByUid(unref(sheetId));
  });

  return computed(() => {
    return new TextDecoder().decode(
      store.getSheetByUid(unref(sheetId))?.content
    );
  });
};

export const useSheetByName = (name: MaybeRef<string>) => {
  const store = useSheetV1Store();
  const ready = ref(false);
  const sheet = computed(() => store.getSheetByName(unref(name)));
  watch(
    () => unref(name),
    (name) => {
      if (!name) return;
      if (extractSheetUID(name) === String(UNKNOWN_ID)) return;

      ready.value = false;
      store.getOrFetchSheetByName(name).finally(() => {
        ready.value = true;
      });
    },
    { immediate: true }
  );
  return { ready, sheet };
};
