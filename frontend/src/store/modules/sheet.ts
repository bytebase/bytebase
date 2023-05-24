import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import axios, { type AxiosRequestConfig } from "axios";
import {
  Sheet,
  SheetId,
  SheetState,
  SheetPatch,
  Principal,
  ResourceObject,
  Database,
  Project,
  unknown,
  UNKNOWN_ID,
  SheetCreate,
  SheetOrganizerUpsert,
  SheetUpsert,
  SheetPayload,
  MaybeRef,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useDatabaseStore } from "./database";
import { useLegacyProjectStore } from "./project";
import { getDefaultSheetPayloadWithSource } from "@/utils";

function convertSheetPayload(
  resourceObj: ResourceObject,
  includedList: ResourceObject[]
) {
  const payload = {};
  try {
    const payloadJSON = resourceObj.attributes.payload;
    if (typeof payloadJSON === "string") {
      Object.assign(payload, JSON.parse(payloadJSON));
    }
  } catch {
    // nothing
  }

  return payload;
}

function convertSheet(
  sheet: ResourceObject,
  includedList: ResourceObject[]
): Sheet {
  let project = unknown("PROJECT") as Project;
  let database = unknown("DATABASE") as Database;

  const projectId = sheet.attributes.projectId;
  const databaseId = sheet.attributes.databaseId || UNKNOWN_ID;

  const databaseStore = useDatabaseStore();
  const projectStore = useLegacyProjectStore();
  for (const item of includedList || []) {
    if (item.type == "project" && Number(item.id) === Number(projectId)) {
      project = projectStore.convert(item, includedList);
    }
    if (item.type == "database" && Number(item.id) == Number(databaseId)) {
      database = databaseStore.convert(item, includedList);
    }
  }

  const payload = convertSheetPayload(sheet, includedList);

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
    payload: payload as SheetPayload,
  };
}

export const useSheetStore = defineStore("sheet", {
  state: (): SheetState => ({
    sheetList: [],
    sheetById: new Map(),
  }),

  actions: {
    getSheetById(sheetId: SheetId) {
      const sheet = this.sheetById.get(sheetId);
      return sheet ?? unknown("SHEET");
    },
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
    upsertSheet(sheetUpsert: SheetUpsert): Promise<Sheet> {
      if (sheetUpsert.id) {
        return this.patchSheetById({
          id: sheetUpsert.id,
          name: sheetUpsert.name,
          statement: sheetUpsert.statement,
          payload: sheetUpsert.payload,
        });
      }

      return this.createSheet({
        payload: getDefaultSheetPayloadWithSource("BYTEBASE"),
        ...sheetUpsert,
        visibility: "PRIVATE",
        source: "BYTEBASE",
      });
    },
    async createSheet(
      sheetCreate: SheetCreate,
      config?: AxiosRequestConfig
    ): Promise<Sheet> {
      if (sheetCreate.databaseId === UNKNOWN_ID) {
        sheetCreate.databaseId = undefined;
      }

      const attributes: Record<string, any> = { ...sheetCreate };
      if (typeof attributes.payload === "object") {
        attributes.payload = JSON.stringify(attributes.payload);
      }

      const resData = (
        await axios.post(
          `/api/sheet`,
          {
            data: {
              type: "createSheet",
              attributes,
            },
          },
          config
        )
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
    async fetchSheetById(sheetId: SheetId) {
      try {
        const data = (await axios.get(`/api/sheet/${sheetId}`)).data;
        const sheet = convertSheet(data.data, data.included);
        this.setSheetById({
          sheetId: sheet.id,
          sheet: sheet,
        });

        return sheet;
      } catch {
        return unknown("SHEET");
      }
    },
    async getOrFetchSheetById(sheetId: SheetId) {
      const storedSheet = this.sheetById.get(sheetId);
      if (storedSheet && storedSheet.id !== UNKNOWN_ID) {
        return storedSheet;
      }
      return this.fetchSheetById(sheetId);
    },
    async patchSheetById(sheetPatch: SheetPatch): Promise<Sheet> {
      const attributes: Record<string, any> = { ...sheetPatch };
      if (typeof attributes.payload === "object") {
        attributes.payload = JSON.stringify(attributes.payload);
      }

      const resData = (
        await axios.patch(`/api/sheet/${sheetPatch.id}`, {
          data: {
            type: "sheetPatch",
            attributes,
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
  },
});

export const useSheetById = (sheetId: MaybeRef<SheetId>) => {
  const store = useSheetStore();
  watchEffect(async () => {
    await store.getOrFetchSheetById(unref(sheetId));
  });

  return computed(() => {
    return store.getSheetById(unref(sheetId));
  });
};
