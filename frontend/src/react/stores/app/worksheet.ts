import { clone, create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniqBy } from "lodash-es";
import { worksheetServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
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
  WorksheetSchema,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetID } from "@/utils";
import type { AppSliceCreator, WorksheetSlice, WorksheetView } from "./types";

const cacheKey = (uid: string, view: WorksheetView) => `${uid}:${view}`;

/**
 * Zustand port of the legacy Pinia `useWorkSheetStore`. Worksheets are
 * keyed by `${uid}:${view}` so FULL (with statement) and BASIC (list)
 * views coexist, matching the old cache. Related resources (project,
 * database, creator) are hydrated through the sibling app slices rather
 * than the old Pinia stores.
 */
export const createWorksheetSlice: AppSliceCreator<WorksheetSlice> = (
  set,
  get
) => {
  const setCacheEntry = (worksheet: Worksheet, view: WorksheetView) => {
    const uid = extractWorksheetID(worksheet.name);
    if (uid === String(UNKNOWN_ID)) return;
    set((s) => {
      const worksheetsByKey = { ...s.worksheetsByKey };
      // A FULL entry supersedes any stale BASIC entry for the same uid.
      if (view === "FULL") {
        delete worksheetsByKey[cacheKey(uid, "BASIC")];
      }
      worksheetsByKey[cacheKey(uid, view)] = worksheet;
      return { worksheetsByKey };
    });
  };

  const hydrateRelatedResources = async (worksheets: Worksheet[]) => {
    try {
      await Promise.all([
        get().batchFetchProjects(worksheets.map((w) => w.project)),
        get().batchFetchDatabases(worksheets.map((w) => w.database)),
        get().batchGetOrFetchUsers(worksheets.map((w) => w.creator)),
      ]);
    } catch {
      // Best-effort hydration; the worksheet entry is still cached below.
    }
  };

  const worksheetList = (): Worksheet[] =>
    uniqBy(Object.values(get().worksheetsByKey), (w) => w.name);

  const updateCacheWithOrganizer = (organizer: WorksheetOrganizer) => {
    for (const view of ["FULL", "BASIC"] as const) {
      const existing = get().getWorksheetByName(organizer.worksheet, view);
      if (!existing) continue;
      const updated = clone(WorksheetSchema, existing);
      updated.starred = organizer.starred;
      updated.folders = organizer.folders;
      setCacheEntry(updated, view);
    }
  };

  return {
    worksheetsByKey: {},
    worksheetRequests: {},

    getWorksheetByName: (name, view) => {
      const uid = extractWorksheetID(name);
      if (!uid || uid === String(UNKNOWN_ID)) return undefined;
      const byKey = get().worksheetsByKey;
      if (view === undefined) {
        return byKey[cacheKey(uid, "FULL")] ?? byKey[cacheKey(uid, "BASIC")];
      }
      return byKey[cacheKey(uid, view)];
    },

    getOrFetchWorksheetByName: async (name, silent = false) => {
      const uid = extractWorksheetID(name);
      if (uid.startsWith("-") || !uid) return undefined;

      const cached = get().worksheetsByKey[cacheKey(uid, "FULL")];
      if (cached) return cached;

      const pending = get().worksheetRequests[uid];
      if (pending) return pending;

      const promise = (async () => {
        try {
          const response = await worksheetServiceClientConnect.getWorksheet(
            createProto(GetWorksheetRequestSchema, { name }),
            {
              contextValues: createContextValues().set(
                silentContextKey,
                silent
              ),
            }
          );
          await hydrateRelatedResources([response]);
          setCacheEntry(response, "FULL");
          return response;
        } catch {
          return undefined;
        } finally {
          set((s) => {
            const { [uid]: _removed, ...worksheetRequests } =
              s.worksheetRequests;
            return { worksheetRequests };
          });
        }
      })();

      set((s) => ({
        worksheetRequests: { ...s.worksheetRequests, [uid]: promise },
      }));
      return promise;
    },

    fetchWorksheetList: async (parent, filter) => {
      const response = await worksheetServiceClientConnect.searchWorksheets(
        createProto(SearchWorksheetsRequestSchema, { parent, filter })
      );
      await hydrateRelatedResources(response.worksheets);
      for (const worksheet of response.worksheets) {
        setCacheEntry(worksheet, "BASIC");
      }
      return response.worksheets;
    },

    createWorksheet: async (worksheet) => {
      const fullWorksheet = worksheet.name
        ? worksheet
        : clone(WorksheetSchema, worksheet);
      const response = await worksheetServiceClientConnect.createWorksheet(
        createProto(CreateWorksheetRequestSchema, {
          parent: fullWorksheet.project,
          worksheet: fullWorksheet,
        })
      );
      setCacheEntry(response, "FULL");
      return response;
    },

    patchWorksheet: async (worksheet, updateMask, signal) => {
      if (!worksheet.name) return undefined;
      const response = await worksheetServiceClientConnect.updateWorksheet(
        createProto(UpdateWorksheetRequestSchema, {
          worksheet,
          updateMask: { paths: updateMask },
        }),
        { signal }
      );
      setCacheEntry(response, "FULL");
      return response;
    },

    deleteWorksheetByName: async (name) => {
      await worksheetServiceClientConnect.deleteWorksheet(
        createProto(DeleteWorksheetRequestSchema, { name })
      );
      const uid = extractWorksheetID(name);
      set((s) => {
        const {
          [cacheKey(uid, "FULL")]: _f,
          [cacheKey(uid, "BASIC")]: _b,
          ...worksheetsByKey
        } = s.worksheetsByKey;
        return { worksheetsByKey };
      });
    },

    upsertWorksheetOrganizer: async (organizer, updateMask) => {
      const response =
        await worksheetServiceClientConnect.updateWorksheetOrganizer(
          createProto(UpdateWorksheetOrganizerRequestSchema, {
            organizer: createProto(WorksheetOrganizerSchema, {
              ...organizer,
            } as WorksheetOrganizer),
            updateMask: { paths: updateMask },
          })
        );
      updateCacheWithOrganizer(response);
    },

    batchUpsertWorksheetOrganizers: async (requests) => {
      const response =
        await worksheetServiceClientConnect.batchUpdateWorksheetOrganizer(
          createProto(BatchUpdateWorksheetOrganizerRequestSchema, {
            requests: requests.map((request) =>
              createProto(UpdateWorksheetOrganizerRequestSchema, {
                organizer: createProto(WorksheetOrganizerSchema, {
                  ...request.organizer,
                } as WorksheetOrganizer),
                updateMask: { paths: request.updateMask },
              })
            ),
          })
        );
      response.worksheetOrganizers.map(updateCacheWithOrganizer);
    },

    myWorksheetList: () => {
      const email = get().currentUser?.email ?? "";
      return worksheetList().filter((w) => w.creator === `users/${email}`);
    },

    sharedWorksheetList: () => {
      const email = get().currentUser?.email ?? "";
      return worksheetList().filter((w) => w.creator !== `users/${email}`);
    },
  };
};
