import { uniqBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { worksheetServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import {
  Worksheet,
  WorksheetOrganizer,
} from "@/types/proto/v1/worksheet_service";
import {
  extractWorksheetUID,
  getSheetStatement,
  isWorksheetReadableV1,
  getStatementSize,
} from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { useTabStore } from "../tab";
import { getUserEmailFromIdentifier } from "./common";

type WorksheetView = "FULL" | "BASIC";
type WorksheetCacheKey = [string /* uid */, WorksheetView];

export const useWorkSheetStore = defineStore("worksheet_v1", () => {
  const cacheByUID = useCache<WorksheetCacheKey, Worksheet | undefined>(
    "bb.worksheet.by-uid"
  );

  // Getters
  const sheetList = computed(() => {
    const sheetList = Array.from(cacheByUID.entityCacheMap.values())
      .map((entry) => entry.entity)
      .filter((sheet): sheet is Worksheet => sheet !== undefined);
    return uniqBy(sheetList, (sheet) => sheet.name);
  });
  const mySheetList = computed(() => {
    const me = useCurrentUserV1();
    return sheetList.value.filter((sheet) => {
      return sheet.creator === `users/${me.value.email}`;
    });
  });
  const sharedSheetList = computed(() => {
    const me = useCurrentUserV1();
    return sheetList.value.filter((sheet) => {
      return sheet.creator !== `users/${me.value.email}`;
    });
  });
  const starredSheetList = computed(() => {
    return sheetList.value.filter((sheet) => {
      return sheet.starred;
    });
  });

  // Utilities
  const setCache = (worksheet: Worksheet, view: WorksheetView) => {
    const uid = extractWorksheetUID(worksheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    if (view === "FULL") {
      // A FULL version should override BASIC version
      cacheByUID.invalidateEntity([uid, "BASIC"]);
    }
    cacheByUID.setEntity([uid, view], worksheet);
  };
  const setListCache = (sheets: Worksheet[]) => {
    sheets.forEach((sheet) => setCache(sheet, "BASIC"));
  };

  // CRUD
  const createSheet = async (worksheet: Partial<Worksheet>) => {
    const created = await worksheetServiceClient.createWorksheet({
      worksheet,
    });
    setCache(created, "FULL");
    return created;
  };

  /**
   *
   * @param name
   * @param view undefined to any (FULL -> BASIC)
   * @returns
   */
  const getSheetByName = (
    name: string,
    view: WorksheetView | undefined = undefined
  ) => {
    const uid = extractWorksheetUID(name);
    if (!uid || uid === String(UNKNOWN_ID)) {
      return undefined;
    }
    if (view === undefined) {
      return (
        cacheByUID.getEntity([uid, "FULL"]) ??
        cacheByUID.getEntity([uid, "BASIC"])
      );
    }
    return cacheByUID.getEntity([uid, view]);
  };
  const fetchSheetByName = async (name: string) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      const sheet = await worksheetServiceClient.getWorksheet({
        name,
      });
      return sheet;
    } catch {
      return undefined;
    }
  };
  const getOrFetchSheetByName = async (name: string) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    const entity = cacheByUID.getEntity([uid, "FULL"]);
    if (entity) {
      return entity;
    }
    const request = cacheByUID.getRequest([uid, "FULL"]);
    if (request) {
      return request;
    }

    const promise = fetchSheetByName(name);
    cacheByUID.setRequest([uid, "FULL"], promise);
    promise.then((sheet) => {
      if (!sheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        cacheByUID.invalidateRequest([uid, "FULL"]);
      }
    });
    return promise;
  };

  const fetchMySheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator = users/${me.value.email}`,
    });
    setListCache(worksheets);
    return worksheets;
  };
  const fetchSharedSheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator != users/${me.value.email}`,
    });
    setListCache(worksheets);
    return worksheets;
  };
  const fetchStarredSheetList = async () => {
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `starred = true`,
    });
    setListCache(worksheets);
    return worksheets;
  };

  const patchSheet = async (
    worksheet: Partial<Worksheet>,
    updateMask: string[]
  ) => {
    if (!worksheet.name) return;
    const updated = await worksheetServiceClient.updateWorksheet({
      worksheet,
      updateMask,
    });
    setCache(updated, "FULL");
    return updated;
  };

  const deleteSheetByName = async (name: string) => {
    await worksheetServiceClient.deleteWorksheet({ name });
    const uid = extractWorksheetUID(name);
    cacheByUID.invalidateEntity([uid, "FULL"]);
    cacheByUID.invalidateEntity([uid, "BASIC"]);
  };

  const upsertSheetOrganizer = async (
    organizer: Pick<WorksheetOrganizer, "worksheet" | "starred">
  ) => {
    await worksheetServiceClient.updateWorksheetOrganizer({
      organizer,
      // for now we only support change the `starred` field.
      updateMask: ["starred"],
    });

    // Update local sheet values
    const fullViewWorksheet = getSheetByName(organizer.worksheet, "FULL");
    if (fullViewWorksheet) {
      fullViewWorksheet.starred = organizer.starred;
    }
    const basicViewWorksheet = getSheetByName(organizer.worksheet, "BASIC");
    if (basicViewWorksheet) {
      basicViewWorksheet.starred = organizer.starred;
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
    fetchMySheetList,
    fetchSharedSheetList,
    fetchStarredSheetList,
    patchSheet,
    deleteSheetByName,
    upsertSheetOrganizer,
  };
});

export const useWorkSheetAndTabStore = defineStore("worksheet_and_tab", () => {
  const tabStore = useTabStore();
  const sheetStore = useWorkSheetStore();
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
    return getUserEmailFromIdentifier(sheet.creator) === me.value.email;
  });

  const isReadOnly = computed(() => {
    const sheet = currentSheet.value;

    // We don't have a selected sheet, we've got nothing to edit.
    if (!sheet) {
      return false;
    }

    // Incomplete sheets should be read-only. e.g. 100MB sheet from issue task.、
    const statement = getSheetStatement(sheet);
    if (getStatementSize(statement).ne(sheet.contentSize)) {
      return true;
    }

    return !isWorksheetReadableV1(sheet);
  });

  return { currentSheet, isCreator, isReadOnly };
});
