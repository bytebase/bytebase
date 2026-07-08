import { create as createProto } from "@bufbuild/protobuf";
import { sheetServiceClientConnect } from "@/connect";
import {
  CreateSheetRequestSchema,
  GetSheetRequestSchema,
  type Sheet,
} from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetUID, isSheetContentComplete } from "@/utils/v1/sheet";
import type { AppSliceCreator, SheetSlice } from "./types";
import { toError } from "./utils";

function isValidSheetName(name: string): boolean {
  if (typeof name !== "string") return false;
  const uid = extractSheetUID(name);
  return Boolean(uid) && uid !== "-1" && !uid.startsWith("-");
}

export const createSheetSlice: AppSliceCreator<SheetSlice> = (set, get) => ({
  sheetsByName: {},
  sheetRequests: {},
  sheetErrorsByName: {},

  fetchSheet: async (name, raw = false) => {
    if (!isValidSheetName(name)) return undefined;
    const existing = get().sheetsByName[name];
    // Sheets are immutable, so a complete cached copy satisfies raw consumers
    // too; a truncated preview only satisfies non-raw ones.
    if (existing && (!raw || isSheetContentComplete(existing))) {
      return existing;
    }
    const pending = get().sheetRequests[name];
    // A pending raw request satisfies everyone; a pending preview request may
    // resolve to truncated content, so a raw consumer can't join it.
    if (pending && (pending.raw || !raw)) return pending.request;

    const request = sheetServiceClientConnect
      .getSheet(createProto(GetSheetRequestSchema, { name, raw }))
      .then((sheet: Sheet) => {
        set((state) => {
          const { [name]: _, ...sheetRequests } = state.sheetRequests;
          const cached = state.sheetsByName[sheet.name];
          // Never replace a complete cached sheet with a truncated preview
          // (a slower non-raw request may resolve after a raw one).
          const next =
            cached &&
            isSheetContentComplete(cached) &&
            !isSheetContentComplete(sheet)
              ? cached
              : sheet;
          return {
            sheetsByName: {
              ...state.sheetsByName,
              [sheet.name]: next,
            },
            sheetErrorsByName: {
              ...state.sheetErrorsByName,
              [name]: undefined,
            },
            sheetRequests,
          };
        });
        return sheet;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...sheetRequests } = state.sheetRequests;
          return {
            sheetErrorsByName: {
              ...state.sheetErrorsByName,
              [name]: toError(error),
            },
            sheetRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      sheetRequests: {
        ...state.sheetRequests,
        [name]: { raw, request },
      },
    }));
    return request;
  },

  createSheet: async (parent, sheet) => {
    const response = await sheetServiceClientConnect.createSheet(
      createProto(CreateSheetRequestSchema, { parent, sheet })
    );
    set((state) => ({
      sheetsByName: {
        ...state.sheetsByName,
        [response.name]: response,
      },
    }));
    return response;
  },

  // Synchronous cache read (undefined on miss) — mirrors the Pinia
  // `getSheetByName`.
  getSheetByName: (name) => get().sheetsByName[name],

  // Cache-first fetch by name — `fetchSheet` already checks the cache and
  // guards invalid names, so this is a thin alias matching the Pinia API.
  getOrFetchSheetByName: (name) => get().fetchSheet(name),
});
