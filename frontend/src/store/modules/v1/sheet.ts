import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { create } from "@bufbuild/protobuf";
import { sheetServiceClientConnect } from "@/grpcweb";
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { 
  CreateSheetRequestSchema,
  GetSheetRequestSchema,
  UpdateSheetRequestSchema
} from "@/types/proto-es/v1/sheet_service_pb";
import { convertNewSheetToOld, convertOldSheetToNew } from "@/utils/v1/sheet-conversions";
import { extractSheetUID, getSheetStatement } from "@/utils";
import { convertEngineToOld } from "@/utils/v1/common-conversions";

export type SheetView = "FULL" | "BASIC";
type SheetCacheKey = [string /* uid */, SheetView];

export const useSheetV1Store = defineStore("sheet_v1", () => {
  const cacheByUID = useCache<SheetCacheKey, Sheet | undefined>(
    "bb.sheet.by-uid"
  );

  // Utilities
  const removeLocalSheet = (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-")) {
      cacheByUID.invalidateEntity([uid, "FULL"]);
    }
  };
  const setCache = (sheet: Sheet, view: SheetView) => {
    const uid = extractSheetUID(sheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    if (view === "FULL") {
      // A FULL version should override BASIC version
      cacheByUID.invalidateEntity([uid, "BASIC"]);
    }
    cacheByUID.setEntity([uid, view], sheet);
  };

  // CRUD
  const createSheet = async (parent: string, sheet: Partial<Sheet>) => {
    if (!sheet.engine || sheet.engine === convertEngineToOld(Engine.ENGINE_UNSPECIFIED)) {
      const engineStr = sheet.engine || "<undefined>";
      console.warn(
        `[SheetService.CreateSheet] sheet.engine unspecified: ${engineStr}`
      );

      sheet.engine = convertEngineToOld(Engine.ENGINE_UNSPECIFIED);
    }
    const fullSheet = Sheet.fromPartial(sheet);
    const request = create(CreateSheetRequestSchema, {
      parent,
      sheet: convertOldSheetToNew(fullSheet),
    });
    const response = await sheetServiceClientConnect.createSheet(request);
    const created = convertNewSheetToOld(response);
    setCache(created, "FULL");
    if (sheet.name) {
      removeLocalSheet(sheet.name);
    }
    return created;
  };

  /**
   *
   * @param uid
   * @param view default undefined to any (FULL -> BASIC)
   * @returns
   */
  const getSheetByUID = (uid: string, view?: SheetView) => {
    if (view === undefined) {
      return (
        cacheByUID.getEntity([uid, "FULL"]) ??
        cacheByUID.getEntity([uid, "BASIC"])
      );
    }
    return cacheByUID.getEntity([uid, view]);
  };
  const getSheetRequestByUID = (uid: string, view?: SheetView) => {
    if (view === undefined) {
      return (
        cacheByUID.getRequest([uid, "FULL"]) ??
        cacheByUID.getRequest([uid, "BASIC"])
      );
    }
    return cacheByUID.getRequest([uid, view]);
  };
  /**
   *
   * @param name
   * @param view default undefined to any (FULL -> BASIC)
   * @returns
   */
  const getSheetByName = (name: string, view?: SheetView) => {
    const uid = extractSheetUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    return getSheetByUID(uid, view);
  };
  const fetchSheetByName = async (name: string, view: SheetView) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      console.debug("[fetchSheetByName]", name, view);
      const request = create(GetSheetRequestSchema, {
        name,
        raw: view === "FULL",
      });
      const response = await sheetServiceClientConnect.getSheet(request);
      const sheet = convertNewSheetToOld(response);
      return sheet;
    } catch {
      return undefined;
    }
  };
  const getOrFetchSheetByName = async (name: string, view?: SheetView) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    const entity = getSheetByUID(uid, view);
    if (entity) {
      return entity;
    }
    const request = getSheetRequestByUID(uid, view);
    if (request) {
      return request;
    }

    const promise = fetchSheetByName(name, view ?? "BASIC");
    cacheByUID.setRequest([uid, view ?? "BASIC"], promise);
    promise.then((sheet) => {
      if (!sheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        cacheByUID.invalidateRequest([uid, view ?? "BASIC"]);
        return;
      }
    });
    return promise;
  };
  const getOrFetchSheetByUID = async (uid: string, view?: SheetView) => {
    return getOrFetchSheetByName(`projects/-/sheets/${uid}`, view);
  };

  const patchSheetContent = async (sheet: Partial<Sheet>) => {
    if (!sheet.name) return;
    const fullSheet = Sheet.fromPartial(sheet);
    const request = create(UpdateSheetRequestSchema, {
      sheet: convertOldSheetToNew(fullSheet),
      updateMask: { paths: ["content"] },
    });
    const response = await sheetServiceClientConnect.updateSheet(request);
    const updated = convertNewSheetToOld(response);
    setCache(updated, "FULL");
    return updated;
  };

  return {
    createSheet,
    getSheetByName,
    fetchSheetByName,
    getOrFetchSheetByName,
    getSheetByUID,
    getOrFetchSheetByUID,
    patchSheetContent,
  };
});

export const useSheetStatementByUID = (
  uid: MaybeRef<string>,
  view?: MaybeRef<SheetView | undefined>
) => {
  const store = useSheetV1Store();
  watchEffect(async () => {
    await store.getOrFetchSheetByUID(unref(uid), unref(view));
  });

  return computed(() => {
    const sheet = store.getSheetByUID(unref(uid), unref(view));
    if (!sheet) return "";
    return getSheetStatement(sheet);
  });
};
