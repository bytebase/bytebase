import { uniqBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { worksheetServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import type {
  Worksheet,
  WorksheetOrganizer,
} from "@/types/proto/v1/worksheet_service";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import {
  extractWorksheetUID,
  getSheetStatement,
  isWorksheetWritableV1,
  getStatementSize,
} from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { useSQLEditorTabStore } from "../sqlEditor";
import { getUserEmailFromIdentifier } from "./common";
import { useDatabaseV1Store, batchGetOrFetchDatabases } from "./database";
import { useProjectV1Store, batchGetOrFetchProjects } from "./project";

type WorksheetView = "FULL" | "BASIC";
type WorksheetCacheKey = [string /* uid */, WorksheetView];

export const useWorkSheetStore = defineStore("worksheet_v1", () => {
  const cacheByUID = useCache<WorksheetCacheKey, Worksheet | undefined>(
    "bb.worksheet.by-uid"
  );
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();

  // Getters
  const worksheetList = computed(() => {
    const sheetList = Array.from(cacheByUID.entityCacheMap.values())
      .map((entry) => entry.entity)
      .filter((sheet): sheet is Worksheet => sheet !== undefined);
    return uniqBy(sheetList, (sheet) => sheet.name);
  });
  const myWorksheetList = computed(() => {
    const me = useCurrentUserV1();
    return worksheetList.value.filter((worksheet) => {
      return worksheet.creator === `users/${me.value.email}`;
    });
  });
  const sharedWorksheetList = computed(() => {
    const me = useCurrentUserV1();
    return worksheetList.value.filter((worksheet) => {
      return worksheet.creator !== `users/${me.value.email}`;
    });
  });
  const starredWorksheetList = computed(() => {
    return worksheetList.value.filter((worksheet) => {
      return worksheet.starred;
    });
  });

  // Utilities
  const setCache = async (worksheet: Worksheet, view: WorksheetView) => {
    const uid = extractWorksheetUID(worksheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    if (view === "FULL") {
      // A FULL version should override BASIC version
      cacheByUID.invalidateEntity([uid, "BASIC"]);
    }

    await Promise.all([
      projectStore.getOrFetchProjectByName(worksheet.project),
      databaseStore.getOrFetchDatabaseByName(worksheet.database),
    ]);
    cacheByUID.setEntity([uid, view], worksheet);
  };
  const setListCache = async (worksheets: Worksheet[]) => {
    await batchGetOrFetchProjects(
      worksheets.map((worksheet) => worksheet.project)
    );
    await batchGetOrFetchDatabases(
      worksheets.map((worksheet) => worksheet.database)
    );
    for (const worksheet of worksheets) {
      await setCache(worksheet, "BASIC");
    }
  };

  // CRUD
  const createWorksheet = async (worksheet: Partial<Worksheet>) => {
    const created = await worksheetServiceClient.createWorksheet({
      worksheet,
    });
    await setCache(created, "FULL");
    return created;
  };

  /**
   *
   * @param name
   * @param view undefined to any (FULL -> BASIC)
   * @returns
   */
  const getWorksheetByName = (
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
  const fetchWorksheetByName = async (name: string) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      const worksheet = await worksheetServiceClient.getWorksheet({
        name,
      });
      return worksheet;
    } catch {
      return undefined;
    }
  };
  const getOrFetchWorksheetByName = async (name: string) => {
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

    const promise = fetchWorksheetByName(name);
    cacheByUID.setRequest([uid, "FULL"], promise);
    promise.then((worksheet) => {
      if (!worksheet) {
        // If the request failed
        // remove the request cache entry so we can retry when needed.
        cacheByUID.invalidateRequest([uid, "FULL"]);
      } else {
        return setCache(worksheet, "FULL");
      }
    });
    return promise;
  };

  const fetchMyWorksheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator = users/${me.value.email}`,
    });
    await setListCache(worksheets);
    return worksheets;
  };
  const fetchSharedWorksheetList = async () => {
    const me = useCurrentUserV1();
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `creator != "users/${me.value.email}" && visibility = "${Worksheet_Visibility.VISIBILITY_PROJECT_READ} | ${Worksheet_Visibility.VISIBILITY_PROJECT_WRITE}"`,
    });
    await setListCache(worksheets);
    return worksheets;
  };

  const fetchStarredWorksheetList = async () => {
    const { worksheets } = await worksheetServiceClient.searchWorksheets({
      filter: `starred = true`,
    });
    await setListCache(worksheets);
    return worksheets;
  };

  const patchWorksheet = async (
    worksheet: Partial<Worksheet>,
    updateMask: string[]
  ) => {
    if (!worksheet.name) return;
    const updated = await worksheetServiceClient.updateWorksheet({
      worksheet,
      updateMask,
    });
    await setCache(updated, "FULL");
    return updated;
  };

  const deleteWorksheetByName = async (name: string) => {
    await worksheetServiceClient.deleteWorksheet({ name });
    const uid = extractWorksheetUID(name);
    cacheByUID.invalidateEntity([uid, "FULL"]);
    cacheByUID.invalidateEntity([uid, "BASIC"]);
  };

  const upsertWorksheetOrganizer = async (
    organizer: Pick<WorksheetOrganizer, "worksheet" | "starred">
  ) => {
    await worksheetServiceClient.updateWorksheetOrganizer({
      organizer,
      // for now we only support change the `starred` field.
      updateMask: ["starred"],
    });

    // Update local sheet values
    const fullViewWorksheet = getWorksheetByName(organizer.worksheet, "FULL");
    if (fullViewWorksheet) {
      fullViewWorksheet.starred = organizer.starred;
      await setCache(fullViewWorksheet, "FULL");
    }
    const basicViewWorksheet = getWorksheetByName(organizer.worksheet, "BASIC");
    if (basicViewWorksheet) {
      basicViewWorksheet.starred = organizer.starred;
      await setCache(basicViewWorksheet, "BASIC");
    }
  };

  return {
    myWorksheetList,
    sharedWorksheetList,
    starredWorksheetList,
    createWorksheet,
    getWorksheetByName,
    getOrFetchWorksheetByName,
    fetchMyWorksheetList,
    fetchSharedWorksheetList,
    fetchStarredWorksheetList,
    patchWorksheet,
    deleteWorksheetByName,
    upsertWorksheetOrganizer,
  };
});

export const useWorkSheetAndTabStore = defineStore("worksheet_and_tab", () => {
  const tabStore = useSQLEditorTabStore();
  const worksheetStore = useWorkSheetStore();
  const me = useCurrentUserV1();

  const currentWorksheet = computed(() => {
    const tab = tabStore.currentTab;
    if (!tab) {
      return undefined;
    }
    const { worksheet } = tab;
    if (!worksheet) {
      return undefined;
    }
    return worksheetStore.getWorksheetByName(worksheet);
  });

  const isCreator = computed(() => {
    const worksheet = currentWorksheet.value;
    if (!worksheet) return false;
    return getUserEmailFromIdentifier(worksheet.creator) === me.value.email;
  });

  const isReadOnly = computed(() => {
    const worksheet = currentWorksheet.value;

    // We don't have a selected sheet, we've got nothing to edit.
    if (!worksheet) {
      return false;
    }

    // Incomplete sheets should be read-only. e.g. 100MB sheet from issue task.、
    const statement = getSheetStatement(worksheet);
    if (getStatementSize(statement).ne(worksheet.contentSize)) {
      return true;
    }

    return !isWorksheetWritableV1(worksheet);
  });

  return { currentSheet: currentWorksheet, isCreator, isReadOnly };
});
