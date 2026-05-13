import { create as createProto } from "@bufbuild/protobuf";
import { databaseGroupServiceClientConnect } from "@/connect";
import { isValidDatabaseGroupName } from "@/types/dbGroup";
import {
  type DatabaseGroup,
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { AppSliceCreator, DBGroupSlice } from "./types";

function toError(error: unknown): Error {
  if (error instanceof Error) return error;
  return new Error(String(error));
}

export const createDBGroupSlice: AppSliceCreator<DBGroupSlice> = (
  set,
  get
) => ({
  dbGroupsByName: {},
  dbGroupRequests: {},
  dbGroupErrorsByName: {},

  fetchDBGroup: async (name) => {
    if (!isValidDatabaseGroupName(name)) return undefined;
    const existing = get().dbGroupsByName[name];
    if (existing) return existing;
    const pending = get().dbGroupRequests[name];
    if (pending) return pending;

    const request = databaseGroupServiceClientConnect
      .getDatabaseGroup(createProto(GetDatabaseGroupRequestSchema, { name }))
      .then((group: DatabaseGroup) => {
        set((state) => {
          const { [name]: _, ...dbGroupRequests } = state.dbGroupRequests;
          return {
            dbGroupsByName: {
              ...state.dbGroupsByName,
              [group.name]: group,
            },
            dbGroupErrorsByName: {
              ...state.dbGroupErrorsByName,
              [name]: undefined,
            },
            dbGroupRequests,
          };
        });
        return group;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...dbGroupRequests } = state.dbGroupRequests;
          return {
            dbGroupErrorsByName: {
              ...state.dbGroupErrorsByName,
              [name]: toError(error),
            },
            dbGroupRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      dbGroupRequests: {
        ...state.dbGroupRequests,
        [name]: request,
      },
    }));
    return request;
  },

  listDBGroupsForProject: async (project) => {
    const response = await databaseGroupServiceClientConnect.listDatabaseGroups(
      createProto(ListDatabaseGroupsRequestSchema, { parent: project })
    );
    set((state) => {
      const next = { ...state.dbGroupsByName };
      for (const group of response.databaseGroups) {
        next[group.name] = group;
      }
      return { dbGroupsByName: next };
    });
    return response.databaseGroups;
  },
});
