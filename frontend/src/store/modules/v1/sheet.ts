import { defineStore } from "pinia";
import { computed, ref, unref, watch, watchEffect } from "vue";
import { isEqual, isUndefined } from "lodash-es";

import { sheetServiceClient } from "@/grpcweb";
import { getUserEmailFromIdentifier } from "./common";
import { extractSheetUID, getSheetStatement, isSheetReadableV1 } from "@/utils";
import {
  Sheet,
  SheetOrganizer,
  Sheet_Source,
} from "@/types/proto/v1/sheet_service";
import { UNKNOWN_ID, MaybeRef } from "@/types";
import { useCurrentUserV1 } from "../auth";
import { useTabStore } from "../tab";

const REQUEST_CACHE_BY_UID = new Map<
  string /* uid */,
  Promise<Sheet | undefined>
>();

export const useSheetV1Store = defineStore("sheet_v1", () => {
  const sheetsByName = ref(new Map<string, Sheet>());

  // Getters
  const sheetListWithoutIssueArtifact = computed(() => {
    // Hide those sheets from issue.
    return Array.from(sheetsByName.value.values()).filter(
      (sheet) => sheet.source !== Sheet_Source.SOURCE_BYTEBASE_ARTIFACT
    );
  });
  const mySheetList = computed(() => {
    const me = useCurrentUserV1();
    return sheetListWithoutIssueArtifact.value.filter((sheet) => {
      return sheet.creator === `users/${me.value.email}`;
    });
  });
  const sharedSheetList = computed(() => {
    const me = useCurrentUserV1();
    return sheetListWithoutIssueArtifact.value.filter((sheet) => {
      return sheet.creator !== `users/${me.value.email}`;
    });
  });
  const starredSheetList = computed(() => {
    return sheetListWithoutIssueArtifact.value.filter((sheet) => {
      return sheet.starred;
    });
  });

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
  const fetchSheetByName = async (name: string) => {
    try {
      const sheet = await sheetServiceClient.getSheet({
        name,
      });

      setSheetList([sheet]);
      return sheet;
    } catch {
      return undefined;
    }
  };
  const getOrFetchSheetByName = async (name: string) => {
    const uid = extractSheetUID(name);
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
  const fetchSheetByUID = async (uid: string, raw = false) => {
    try {
      const name = `projects/-/sheets/${uid}`;
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
  const getOrFetchSheetByUID = async (uid: string) => {
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
  const fetchMySheetList = async () => {
    const me = useCurrentUserV1();
    const { sheets } = await sheetServiceClient.searchSheets({
      parent: "projects/-",
      filter: `creator = users/${me.value.email}`,
    });
    setSheetList(sheets);
    return sheets;
  };
  const fetchSharedSheetList = async () => {
    const me = useCurrentUserV1();
    const { sheets } = await sheetServiceClient.searchSheets({
      parent: "projects/-",
      filter: `creator != users/${me.value.email}`,
    });
    setSheetList(sheets);
    return sheets;
  };
  const fetchStarredSheetList = async () => {
    const { sheets } = await sheetServiceClient.searchSheets({
      parent: "projects/-",
      filter: "starred = true",
    });
    setSheetList(sheets);
    return sheets;
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

  const deleteSheetByName = async (name: string) => {
    await sheetServiceClient.deleteSheet({ name });
    sheetsByName.value.delete(name);
  };

  // Other functions
  const syncSheetFromVCS = async (parent: string) => {
    await sheetServiceClient.syncSheets({
      parent,
    });
  };
  const upsertSheetOrganizer = async (
    organizer: Pick<SheetOrganizer, "sheet" | "starred">
  ) => {
    await sheetServiceClient.updateSheetOrganizer({
      organizer,
      // for now we only support change the `starred` field.
      updateMask: ["starred"],
    });

    // Update local sheet values
    const sheet = getSheetByName(organizer.sheet);
    if (sheet) {
      sheet.starred = organizer.starred;
    }
  };

  return {
    mySheetList,
    sharedSheetList,
    starredSheetList,
    createSheet,
    getSheetByName,
    fetchSheetByName,
    getOrFetchSheetByName,
    getSheetByUID,
    fetchSheetByUID,
    getOrFetchSheetByUID,
    fetchMySheetList,
    fetchSharedSheetList,
    fetchStarredSheetList,
    patchSheet,
    deleteSheetByName,
    syncSheetFromVCS,
    upsertSheetOrganizer,
  };
});

export const useSheetAndTabStore = defineStore("sheet_and_tab", () => {
  const tabStore = useTabStore();
  const sheetStore = useSheetV1Store();
  const me = useCurrentUserV1();

  const currentSheet = computed(() => {
    const tab = tabStore.currentTab;
    const name = tab.sheetName;
    if (!name) {
      return undefined;
    }
    return sheetStore.getSheetByName(name);
  });

  const isCreator = computed(() => {
    const sheet = currentSheet.value;
    if (!sheet) return false;
    return getUserEmailFromIdentifier(sheet.name) === me.value.email;
  });

  const isReadOnly = computed(() => {
    const sheet = currentSheet.value;

    // We don't have a selected sheet, we've got nothing to edit.
    if (!sheet) {
      return false;
    }

    // Incomplete sheets should be read-only. e.g. 100MB sheet from issue task.
    if (sheet.content.length !== sheet.contentSize) {
      return true;
    }

    return !isSheetReadableV1(sheet);
  });

  return { currentSheet, isCreator, isReadOnly };
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
