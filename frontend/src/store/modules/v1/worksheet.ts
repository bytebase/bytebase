import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { worksheetServiceClient } from "@/grpcweb";
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

const REQUEST_CACHE_BY_UID = new Map<
  string /* uid */,
  Promise<Worksheet | undefined>
>();

export const useWorkSheetStore = defineStore("worksheet_v1", () => {
  const sheetsByName = ref(new Map<string, Worksheet>());

  // Getters
  const sheetList = computed(() => {
    // Hide those sheets from issue.
    return Array.from(sheetsByName.value.values());
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
  const removeLocalSheet = (name: string) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-")) {
      sheetsByName.value.delete(name);
    }
  };
  const setSheetList = (sheets: Worksheet[]) => {
    for (const sheet of sheets) {
      sheetsByName.value.set(sheet.name, sheet);
    }
  };

  // CRUD
  const createSheet = async (worksheet: Partial<Worksheet>) => {
    const created = await worksheetServiceClient.createWorksheet({
      worksheet,
    });
    setSheetList([created]);
    if (worksheet.name) {
      removeLocalSheet(worksheet.name);
    }
    return created;
  };

  const getSheetByName = (name: string) => {
    return sheetsByName.value.get(name);
  };
  const fetchSheetByName = async (name: string, raw = false) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      const sheet = await worksheetServiceClient.getWorksheet({
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
    const uid = extractWorksheetUID(name);
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

  const fetchMySheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator = users/${me.value.email}`,
    });
    setSheetList(worksheets);
    return worksheets;
  };
  const fetchSharedSheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator != users/${me.value.email}`,
    });
    setSheetList(worksheets);
    return worksheets;
  };
  const fetchStarredSheetList = async () => {
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `starred = true`,
    });
    setSheetList(worksheets);
    return worksheets;
  };

  const patchSheet = async (
    worksheet: Partial<Worksheet>,
    updateMask: string[] | undefined = undefined
  ) => {
    if (!worksheet.name) return;
    const existed = sheetsByName.value.get(worksheet.name);
    if (!existed) return;
    if (!updateMask) {
      updateMask = getUpdateMaskForSheet(existed, worksheet);
    }
    if (updateMask.length === 0) {
      return existed;
    }
    const updated = await worksheetServiceClient.updateWorksheet({
      worksheet,
      updateMask,
    });
    setSheetList([updated]);
    return updated;
  };

  const deleteSheetByName = async (name: string) => {
    await worksheetServiceClient.deleteWorksheet({ name });
    sheetsByName.value.delete(name);
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
    const sheet = getSheetByName(organizer.worksheet);
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

const getUpdateMaskForSheet = (
  origin: Worksheet,
  update: Partial<Worksheet>
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
  return updateMask;
};
