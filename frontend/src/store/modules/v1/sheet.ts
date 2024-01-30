import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref, unref, watchEffect } from "vue";
import { sheetServiceClient } from "@/grpcweb";
import { UNKNOWN_ID, MaybeRef } from "@/types";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { extractSheetUID, getSheetStatement } from "@/utils";

const REQUEST_CACHE_BY_UID = new Map<
  string /* uid */,
  Promise<Sheet | undefined>
>();

export const useSheetV1Store = defineStore("sheet_v1", () => {
  const sheetsByName = ref(new Map<string, Sheet>());

  // Utilities
  const removeLocalSheet = (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-")) {
      sheetsByName.value.delete(name);
    }
  };
  const setSheetList = (sheets: Sheet[]) => {
    for (const sheet of sheets) {
      sheetsByName.value.set(sheet.name, sheet);
    }
  };

  // CRUD
  const createSheet = async (parent: string, sheet: Partial<Sheet>) => {
    const created = await sheetServiceClient.createSheet({
      parent,
      sheet,
    });
    setSheetList([created]);
    if (sheet.name) {
      removeLocalSheet(sheet.name);
    }
    return created;
  };

  const getSheetByName = (name: string) => {
    return sheetsByName.value.get(name);
  };
  const fetchSheetByName = async (name: string, raw = false) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      const sheet = await sheetServiceClient.getSheet({
        name,
        raw,
      });

      setSheetList([sheet]);
      return sheet;
    } catch {
      return undefined;
    }
  };
  const getOrFetchSheetByName = async (name: string) => {
    const uid = extractSheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    if (uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const existed = getSheetByName(name);
    if (existed) {
      return existed;
    }
    const cached = REQUEST_CACHE_BY_UID.get(uid);
    if (cached) {
      return cached;
    }

    const request = fetchSheetByName(name);
    REQUEST_CACHE_BY_UID.set(uid, request);
    request.then((sheet) => {
      if (!sheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        REQUEST_CACHE_BY_UID.delete(uid);
      }
    });
    return request;
  };
  const getSheetByUID = (uid: string) => {
    for (const [name, sheet] of sheetsByName.value) {
      if (extractSheetUID(name) === uid) {
        return sheet;
      }
    }
  };
  const getOrFetchSheetByUID = async (uid: string) => {
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    if (uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    const existed = getSheetByUID(uid);
    if (existed) {
      return existed;
    }
    const cached = REQUEST_CACHE_BY_UID.get(uid);
    if (cached) {
      return cached;
    }

    const name = `projects/-/sheets/${uid}`;
    const request = fetchSheetByName(name);
    REQUEST_CACHE_BY_UID.set(uid, request);
    request.then((sheet) => {
      if (!sheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        REQUEST_CACHE_BY_UID.delete(uid);
      }
    });
    return request;
  };

  const patchSheet = async (
    sheet: Partial<Sheet>,
    updateMask: string[] | undefined = undefined
  ) => {
    if (!sheet.name) return;
    const existed = sheetsByName.value.get(sheet.name);
    if (!existed) return;
    if (!updateMask) {
      updateMask = getUpdateMaskForSheet(existed, sheet);
    }
    if (updateMask.length === 0) {
      return existed;
    }
    const updated = await sheetServiceClient.updateSheet({
      sheet,
      updateMask,
    });
    setSheetList([updated]);
    return updated;
  };

  return {
    createSheet,
    getSheetByName,
    fetchSheetByName,
    getOrFetchSheetByName,
    getSheetByUID,
    getOrFetchSheetByUID,
    patchSheet,
  };
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
  return updateMask;
};

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
