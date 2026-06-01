import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseGroupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { buildCELExpr } from "@/plugins/cel";
import { databaseGroupNamePrefix } from "@/store/modules/v1/common";
import {
  isValidDatabaseGroupName,
  unknownDatabaseGroup,
} from "@/types/dbGroup";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  CreateDatabaseGroupRequestSchema,
  type DatabaseGroup,
  DatabaseGroupSchema,
  DatabaseGroupView,
  DeleteDatabaseGroupRequestSchema,
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
  UpdateDatabaseGroupRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils";
import type { AppSliceCreator, DBGroupSlice } from "./types";
import { toError } from "./utils";

export const createDBGroupSlice: AppSliceCreator<DBGroupSlice> = (set, get) => {
  // Immutable cache write. A name maps to a single group + the view it was
  // fetched with; overwriting is sufficient (no per-view keying needed). Also
  // clears any stale error so a successful (re)fetch doesn't leave a prior
  // failure visible to consumers reading `dbGroupErrorsByName`.
  const setCache = (group: DatabaseGroup, view: DatabaseGroupView) => {
    set((state) => ({
      dbGroupsByName: { ...state.dbGroupsByName, [group.name]: group },
      dbGroupViewByName: { ...state.dbGroupViewByName, [group.name]: view },
      dbGroupErrorsByName: {
        ...state.dbGroupErrorsByName,
        [group.name]: undefined,
      },
    }));
  };
  const removeCache = (name: string) => {
    set((state) => {
      const { [name]: _g, ...dbGroupsByName } = state.dbGroupsByName;
      const { [name]: _v, ...dbGroupViewByName } = state.dbGroupViewByName;
      return { dbGroupsByName, dbGroupViewByName };
    });
  };
  // Bulk variant of `setCache` — merges every group in one `set()` so a list
  // fetch notifies subscribers once instead of N times.
  const cacheGroups = (groups: DatabaseGroup[], view: DatabaseGroupView) => {
    set((state) => {
      const dbGroupsByName = { ...state.dbGroupsByName };
      const dbGroupViewByName = { ...state.dbGroupViewByName };
      for (const group of groups) {
        dbGroupsByName[group.name] = group;
        dbGroupViewByName[group.name] = view;
      }
      return { dbGroupsByName, dbGroupViewByName };
    });
  };

  return {
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
      const response =
        await databaseGroupServiceClientConnect.listDatabaseGroups(
          createProto(ListDatabaseGroupsRequestSchema, { parent: project })
        );
      cacheGroups(response.databaseGroups, DatabaseGroupView.BASIC);
      return response.databaseGroups;
    },

    getDBGroupByName: (name, view = DatabaseGroupView.UNSPECIFIED) => {
      const { dbGroupsByName, dbGroupViewByName } = get();
      const group = dbGroupsByName[name];
      if (group) {
        const satisfied =
          view === DatabaseGroupView.FULL
            ? dbGroupViewByName[name] === DatabaseGroupView.FULL
            : true; // BASIC/UNSPECIFIED requests accept any cached view
        if (satisfied) return group;
      }
      return unknownDatabaseGroup();
    },

    getOrFetchDBGroupByName: async (name, options) => {
      const {
        skipCache = false,
        silent = false,
        view = DatabaseGroupView.BASIC,
      } = options ?? {};
      if (!skipCache) {
        const cached = get().getDBGroupByName(name, view);
        if (isValidDatabaseGroupName(cached.name)) {
          return cached;
        }
      }
      const group = await databaseGroupServiceClientConnect.getDatabaseGroup(
        createProto(GetDatabaseGroupRequestSchema, { name, view }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      );
      setCache(group, view);
      return group;
    },

    fetchDBGroupListByProjectName: async (projectName, view) => {
      const response =
        await databaseGroupServiceClientConnect.listDatabaseGroups(
          createProto(ListDatabaseGroupsRequestSchema, {
            parent: projectName,
            view,
          })
        );
      cacheGroups(response.databaseGroups, view);
      return response.databaseGroups;
    },

    createDatabaseGroup: async ({
      projectName,
      databaseGroup,
      databaseGroupId,
      validateOnly = false,
    }) => {
      const newDatabaseGroup = createProto(DatabaseGroupSchema, {
        name: databaseGroup.name,
        title: databaseGroup.title,
        databaseExpr: databaseGroup.databaseExpr,
        matchedDatabases: [],
      });
      const created =
        await databaseGroupServiceClientConnect.createDatabaseGroup(
          createProto(CreateDatabaseGroupRequestSchema, {
            parent: projectName,
            databaseGroup: newDatabaseGroup,
            databaseGroupId,
            validateOnly,
          }),
          {
            contextValues: createContextValues().set(
              silentContextKey,
              validateOnly
            ),
          }
        );
      if (!validateOnly) {
        setCache(created, DatabaseGroupView.FULL);
      }
      return created;
    },

    updateDatabaseGroup: async (databaseGroup, updateMask) => {
      const updated =
        await databaseGroupServiceClientConnect.updateDatabaseGroup(
          createProto(UpdateDatabaseGroupRequestSchema, {
            databaseGroup,
            updateMask: { paths: updateMask },
          })
        );
      setCache(updated, DatabaseGroupView.FULL);
      return updated;
    },

    deleteDatabaseGroup: async (name) => {
      await databaseGroupServiceClientConnect.deleteDatabaseGroup(
        createProto(DeleteDatabaseGroupRequestSchema, { name })
      );
      removeCache(name);
    },

    fetchDatabaseGroupMatchList: async ({ projectName, expr }) => {
      const celexpr = await buildCELExpr(expr);
      if (!celexpr) {
        return [];
      }
      const celStrings = await batchConvertParsedExprToCELString([celexpr]);
      const expression = celStrings[0] || "true";
      const validateOnlyResourceId = `creating-database-group-${Date.now()}`;
      const result = await get().createDatabaseGroup({
        projectName,
        databaseGroup: createProto(DatabaseGroupSchema, {
          name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
          title: validateOnlyResourceId,
          databaseExpr: createProto(ExprSchema, { expression }),
        }),
        databaseGroupId: validateOnlyResourceId,
        validateOnly: true,
      });
      return result.matchedDatabases.map((item) => item.name);
    },
  };
};
