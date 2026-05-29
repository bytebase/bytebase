import { create as createProto } from "@bufbuild/protobuf";
import { databaseGroupServiceClientConnect } from "@/connect";
import { isValidDatabaseGroupName } from "@/types/dbGroup";
import {
  type DatabaseGroup,
  DatabaseGroupView,
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import type { AppSliceCreator, DBGroupSlice } from "./types";
import { toError } from "./utils";

export const createDBGroupSlice: AppSliceCreator<DBGroupSlice> = (
  set,
  get
) => ({
  dbGroupsByName: {},
  dbGroupViewByName: {},
  dbGroupRequests: {},
  dbGroupErrorsByName: {},

  fetchDBGroup: async (name, view = DatabaseGroupView.BASIC) => {
    if (!isValidDatabaseGroupName(name)) return undefined;
    const existing = get().dbGroupsByName[name];
    // A FULL request needs `matchedDatabases`; only reuse the cache when it
    // already holds the FULL view (a BASIC request is satisfied by either).
    if (
      existing &&
      (view !== DatabaseGroupView.FULL ||
        get().dbGroupViewByName[name] === DatabaseGroupView.FULL)
    ) {
      return existing;
    }
    const pending = get().dbGroupRequests[name];
    if (pending) return pending;

    const request = databaseGroupServiceClientConnect
      .getDatabaseGroup(
        createProto(GetDatabaseGroupRequestSchema, { name, view })
      )
      .then((group: DatabaseGroup) => {
        set((state) => {
          const { [name]: _, ...dbGroupRequests } = state.dbGroupRequests;
          return {
            dbGroupsByName: {
              ...state.dbGroupsByName,
              [group.name]: group,
            },
            dbGroupViewByName: {
              ...state.dbGroupViewByName,
              [group.name]: view,
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
      const nextView = { ...state.dbGroupViewByName };
      for (const group of response.databaseGroups) {
        next[group.name] = group;
        nextView[group.name] = DatabaseGroupView.BASIC;
      }
      return { dbGroupsByName: next, dbGroupViewByName: nextView };
    });
    return response.databaseGroups;
  },
});
