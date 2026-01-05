import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { sheetServiceClientConnect } from "@/connect";
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID } from "@/types";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  CreateSheetRequestSchema,
  GetSheetRequestSchema,
  SheetSchema,
} from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetUID, getSheetStatement } from "@/utils";

export const useSheetV1Store = defineStore("sheet_v1", () => {
  const cacheByUID = useCache<[string], Sheet | undefined>("bb.sheet.by-uid");

  // Utilities
  const removeLocalSheet = (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-")) {
      cacheByUID.invalidateEntity([uid]);
    }
  };
  const setCache = (sheet: Sheet) => {
    const uid = extractSheetUID(sheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    cacheByUID.setEntity([uid], sheet);
  };

  // CRUD
  const createSheet = async (parent: string, sheet: Partial<Sheet>) => {
    const fullSheet = create(SheetSchema, {
      name: sheet.name || "",
      content: sheet.content || new Uint8Array(),
      contentSize: sheet.contentSize || BigInt(0),
    });
    const request = create(CreateSheetRequestSchema, {
      parent,
      sheet: fullSheet,
    });
    const response = await sheetServiceClientConnect.createSheet(request);
    setCache(response);
    if (sheet.name) {
      removeLocalSheet(sheet.name);
    }
    return response;
  };

  const getSheetByUID = (uid: string) => {
    return cacheByUID.getEntity([uid]);
  };
  const getSheetRequestByUID = (uid: string) => {
    return cacheByUID.getRequest([uid]);
  };
  const getSheetByName = (name: string) => {
    const uid = extractSheetUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    return getSheetByUID(uid);
  };
  const fetchSheetByName = async (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      console.debug("[fetchSheetByName]", name);
      const request = create(GetSheetRequestSchema, {
        name,
        raw: false,
      });
      const response = await sheetServiceClientConnect.getSheet(request);
      return response;
    } catch {
      return undefined;
    }
  };
  const getOrFetchSheetByName = async (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    const entity = getSheetByUID(uid);
    if (entity) {
      return entity;
    }
    const request = getSheetRequestByUID(uid);
    if (request) {
      return request;
    }

    const promise = fetchSheetByName(name);
    cacheByUID.setRequest([uid], promise);
    promise.then((sheet) => {
      if (!sheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        cacheByUID.invalidateRequest([uid]);
        return;
      }
    });
    return promise;
  };
  const getOrFetchSheetByUID = async (uid: string) => {
    return getOrFetchSheetByName(`projects/-/sheets/${uid}`);
  };

  return {
    createSheet,
    getSheetByName,
    fetchSheetByName,
    getOrFetchSheetByName,
    getSheetByUID,
    getOrFetchSheetByUID,
  };
});

export const useSheetStatementByUID = (uid: MaybeRef<string>) => {
  const store = useSheetV1Store();
  watchEffect(async () => {
    await store.getOrFetchSheetByUID(unref(uid));
  });

  return computed(() => {
    const sheet = store.getSheetByUID(unref(uid));
    if (!sheet) return "";
    return getSheetStatement(sheet);
  });
};
