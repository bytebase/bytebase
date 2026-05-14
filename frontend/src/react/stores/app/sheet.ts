import { create as createProto } from "@bufbuild/protobuf";
import { sheetServiceClientConnect } from "@/connect";
import {
  CreateSheetRequestSchema,
  GetSheetRequestSchema,
  type Sheet,
} from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetUID } from "@/utils/v1/sheet";
import type { AppSliceCreator, SheetSlice } from "./types";

function toError(error: unknown): Error {
  if (error instanceof Error) return error;
  return new Error(String(error));
}

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
    if (!raw) {
      const existing = get().sheetsByName[name];
      if (existing) return existing;
    }
    const pending = get().sheetRequests[name];
    if (pending) return pending;

    const request = sheetServiceClientConnect
      .getSheet(createProto(GetSheetRequestSchema, { name, raw }))
      .then((sheet: Sheet) => {
        set((state) => {
          const { [name]: _, ...sheetRequests } = state.sheetRequests;
          return {
            sheetsByName: {
              ...state.sheetsByName,
              [sheet.name]: sheet,
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
        [name]: request,
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
});
