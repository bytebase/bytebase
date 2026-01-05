import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniqBy } from "lodash-es";
import { defineStore } from "pinia";
import { computed } from "vue";
import { worksheetServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { useCache } from "@/store/cache";
import { UNKNOWN_ID } from "@/types";
import type {
  Worksheet,
  WorksheetOrganizer,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  BatchUpdateWorksheetOrganizerRequestSchema,
  CreateWorksheetRequestSchema,
  DeleteWorksheetRequestSchema,
  GetWorksheetRequestSchema,
  SearchWorksheetsRequestSchema,
  UpdateWorksheetOrganizerRequestSchema,
  UpdateWorksheetRequestSchema,
  WorksheetOrganizerSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import {
  extractWorksheetUID,
  getSheetStatement,
  getStatementSize,
  isWorksheetWritableV1,
} from "@/utils";
import { useSQLEditorTabStore } from "../sqlEditor";
import { useUserStore } from "../user";
import { useCurrentUserV1 } from "./auth";
import { extractUserId } from "./common";
import { useDatabaseV1Store } from "./database";
import { useProjectV1Store } from "./project";

type WorksheetView = "FULL" | "BASIC";
type WorksheetCacheKey = [string /* uid */, WorksheetView];

export const useWorkSheetStore = defineStore("worksheet_v1", () => {
  const cacheByUID = useCache<WorksheetCacheKey, Worksheet | undefined>(
    "bb.worksheet.by-uid"
  );
  const projectStore = useProjectV1Store();
  const databaseStore = useDatabaseV1Store();
  const userStore = useUserStore();

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

  // Utilities
  const setCacheEntry = (worksheet: Worksheet, view: WorksheetView) => {
    const uid = extractWorksheetUID(worksheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    if (view === "FULL") {
      cacheByUID.invalidateEntity([uid, "BASIC"]);
    }
    cacheByUID.setEntity([uid, view], worksheet);
  };

  const setCache = async (worksheet: Worksheet, view: WorksheetView) => {
    try {
      await Promise.all([
        projectStore.getOrFetchProjectByName(worksheet.project),
        databaseStore.getOrFetchDatabaseByName(worksheet.database),
        userStore.getOrFetchUserByIdentifier(worksheet.creator),
      ]);
    } catch {
      // ignore error
    }
    setCacheEntry(worksheet, view);
  };

  const setListCache = async (worksheets: Worksheet[]) => {
    await Promise.all([
      projectStore.batchGetOrFetchProjects(
        worksheets.map((worksheet) => worksheet.project)
      ),
      databaseStore.batchGetOrFetchDatabases(
        worksheets.map((worksheet) => worksheet.database)
      ),
      userStore.batchGetOrFetchUsers(
        worksheets.map((worksheet) => worksheet.creator)
      ),
    ]);
    for (const worksheet of worksheets) {
      setCacheEntry(worksheet, "BASIC");
    }
  };

  // CRUD
  const createWorksheet = async (worksheet: Worksheet) => {
    const fullWorksheet = worksheet.name
      ? worksheet
      : { ...worksheet, name: "" };
    const request = create(CreateWorksheetRequestSchema, {
      worksheet: fullWorksheet,
    });
    const response =
      await worksheetServiceClientConnect.createWorksheet(request);
    setCacheEntry(response, "FULL");
    return response;
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
  const fetchWorksheetByName = async (
    name: string,
    silent: boolean = false
  ) => {
    const uid = extractWorksheetUID(name);
    if (uid.startsWith("-") || !uid) {
      return undefined;
    }
    try {
      const request = create(GetWorksheetRequestSchema, {
        name,
      });
      const response = await worksheetServiceClientConnect.getWorksheet(
        request,
        {
          contextValues: createContextValues().set(silentContextKey, silent),
        }
      );
      return response;
    } catch {
      return undefined;
    }
  };
  const getOrFetchWorksheetByName = async (
    name: string,
    silent: boolean = false
  ) => {
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

    const promise = fetchWorksheetByName(name, silent);
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

  const fetchWorksheetList = async (filter: string) => {
    const request = create(SearchWorksheetsRequestSchema, {
      filter,
    });
    const response =
      await worksheetServiceClientConnect.searchWorksheets(request);
    await setListCache(response.worksheets);
    return response.worksheets;
  };

  const patchWorksheet = async (worksheet: Worksheet, updateMask: string[]) => {
    if (!worksheet.name) return;
    const request = create(UpdateWorksheetRequestSchema, {
      worksheet: worksheet,
      updateMask: { paths: updateMask },
    });
    const response =
      await worksheetServiceClientConnect.updateWorksheet(request);
    setCacheEntry(response, "FULL");
    return response;
  };

  const deleteWorksheetByName = async (name: string) => {
    const request = create(DeleteWorksheetRequestSchema, { name });
    await worksheetServiceClientConnect.deleteWorksheet(request);
    const uid = extractWorksheetUID(name);
    cacheByUID.invalidateEntity([uid, "FULL"]);
    cacheByUID.invalidateEntity([uid, "BASIC"]);
  };

  const updateWorksheetCacheWithOrganizer = (organizer: WorksheetOrganizer) => {
    // Update local sheet values
    const views: WorksheetView[] = ["FULL", "BASIC"];
    for (const view of views) {
      const worksheet = getWorksheetByName(organizer.worksheet, view);
      if (worksheet) {
        worksheet.starred = organizer.starred;
        worksheet.folders = organizer.folders;
        setCacheEntry(worksheet, view);
      }
    }
  };

  const batchUpsertWorksheetOrganizers = async (
    requests: {
      organizer: Partial<WorksheetOrganizer>;
      updateMask: string[];
    }[]
  ) => {
    const request = create(BatchUpdateWorksheetOrganizerRequestSchema, {
      requests: requests.map((request) =>
        create(UpdateWorksheetOrganizerRequestSchema, {
          organizer: create(WorksheetOrganizerSchema, {
            ...request.organizer,
          } as WorksheetOrganizer),
          updateMask: { paths: request.updateMask },
        })
      ),
    });
    const response =
      await worksheetServiceClientConnect.batchUpdateWorksheetOrganizer(
        request
      );

    response.worksheetOrganizers.map(updateWorksheetCacheWithOrganizer);
  };

  const upsertWorksheetOrganizer = async (
    organizer: Partial<WorksheetOrganizer>,
    updateMask: string[]
  ) => {
    const request = create(UpdateWorksheetOrganizerRequestSchema, {
      organizer: create(WorksheetOrganizerSchema, {
        ...organizer,
      } as WorksheetOrganizer),
      updateMask: { paths: updateMask },
    });
    const response =
      await worksheetServiceClientConnect.updateWorksheetOrganizer(request);
    updateWorksheetCacheWithOrganizer(response);
  };

  return {
    myWorksheetList,
    sharedWorksheetList,
    createWorksheet,
    getWorksheetByName,
    getOrFetchWorksheetByName,
    fetchWorksheetList,
    patchWorksheet,
    deleteWorksheetByName,
    upsertWorksheetOrganizer,
    batchUpsertWorksheetOrganizers,
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
    return extractUserId(worksheet.creator) === me.value.email;
  });

  const isReadOnly = computed(() => {
    const worksheet = currentWorksheet.value;

    // We don't have a selected sheet, we've got nothing to edit.
    if (!worksheet) {
      return false;
    }

    // Incomplete sheets should be read-only. e.g. 100MB sheet from issue task.、
    const statement = getSheetStatement(worksheet);
    if (getStatementSize(statement) !== worksheet.contentSize) {
      return true;
    }

    return !isWorksheetWritableV1(worksheet);
  });

  return { currentSheet: currentWorksheet, isCreator, isReadOnly };
});
