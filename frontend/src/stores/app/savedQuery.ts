import { clone, create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniqBy } from "lodash-es";
import { savedQueryServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { UNKNOWN_ID } from "@/types";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import {
  CreateSavedQueryRequestSchema,
  DeleteSavedQueryRequestSchema,
  GetSavedQueryPolicyRequestSchema,
  GetSavedQueryRequestSchema,
  MoveMySavedQueriesRequestSchema,
  SavedQueryBinding_Level,
  SavedQuerySchema,
  SearchSavedQueriesRequestSchema,
  SearchSavedQueryFoldersRequestSchema,
  SetSavedQueryPolicyRequestSchema,
  UpdateSavedQueryRequestSchema,
  UpdateSavedQueryStarRequestSchema,
} from "@/types/proto-es/v1/saved_query_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import { extractSavedQueryID } from "@/utils/v1/savedQuery";
import { setSavedQueryAccess } from "./savedQueryAccess";
import type { AppSliceCreator, SavedQuerySlice, SavedQueryView } from "./types";

const cacheKey = (uid: string, view: SavedQueryView) => `${uid}:${view}`;

/**
 * Zustand port of the legacy Pinia `useWorkSheetStore`. Saved queries are
 * keyed by `${uid}:${view}` so FULL (with statement) and BASIC (list)
 * views coexist, matching the old cache. Related resources (project,
 * database, creator) are hydrated through the sibling app slices rather
 * than the old Pinia stores.
 */
export const createSavedQuerySlice: AppSliceCreator<SavedQuerySlice> = (
  set,
  get
) => {
  const setCacheEntry = (savedQuery: SavedQuery, view: SavedQueryView) => {
    const uid = extractSavedQueryID(savedQuery.name);
    if (uid === String(UNKNOWN_ID)) return;
    set((s) => {
      const savedQueriesByKey = { ...s.savedQueriesByKey };
      // A FULL entry supersedes any stale BASIC entry for the same uid.
      if (view === "FULL") {
        delete savedQueriesByKey[cacheKey(uid, "BASIC")];
      }
      savedQueriesByKey[cacheKey(uid, view)] = savedQuery;
      return { savedQueriesByKey };
    });
  };

  const hydrateRelatedResources = async (savedQueries: SavedQuery[]) => {
    // A saved query without a connection has `database: ""`. The batch
    // endpoint rejects the whole request on an invalid name, which would
    // drop hydration for every valid database in the same batch, so filter
    // those out first. Dedupe all three to keep the batch payloads minimal.
    const databases = [
      ...new Set(
        savedQueries.map((w) => w.database).filter(isValidDatabaseName)
      ),
    ];
    try {
      await Promise.all([
        get().batchFetchProjects([
          ...new Set(savedQueries.map((w) => w.project)),
        ]),
        get().batchFetchDatabases(databases),
        get().batchGetOrFetchUsers([
          ...new Set(savedQueries.map((w) => w.creator)),
        ]),
      ]);
    } catch {
      // Best-effort hydration; the saved query entry is still cached below.
    }
  };

  // The caller's principals, in the binding format the policy stores. Mirrors
  // the server's principals(u): themselves plus each group they belong to,
  // matched by reference so group membership stays live.
  const callerPrincipals = (): string[] => {
    const user = get().currentUser;
    if (!user?.email) return [];
    return [
      `user:${user.email}`,
      ...user.groups.map((group) => `group:${group.replace(/^groups\//, "")}`),
    ];
  };

  // Wire the cycle-free access layer the shared `@/utils` predicates read.
  setSavedQueryAccess({
    getSavedQueryLevel: (name) =>
      get().savedQueryLevelByName[name] ??
      SavedQueryBinding_Level.LEVEL_UNSPECIFIED,
  });

  return {
    savedQueriesByKey: {},
    savedQueryRequests: {},
    savedQueryLevelByName: {},

    getSavedQueryByName: (name, view) => {
      const uid = extractSavedQueryID(name);
      if (!uid || uid === String(UNKNOWN_ID)) return undefined;
      const byKey = get().savedQueriesByKey;
      if (view === undefined) {
        return byKey[cacheKey(uid, "FULL")] ?? byKey[cacheKey(uid, "BASIC")];
      }
      return byKey[cacheKey(uid, view)];
    },

    getOrFetchSavedQueryByName: async (name, silent = false) => {
      const uid = extractSavedQueryID(name);
      if (uid.startsWith("-") || !uid) return undefined;

      const cached = get().savedQueriesByKey[cacheKey(uid, "FULL")];
      if (cached) return cached;

      const pending = get().savedQueryRequests[uid];
      if (pending) return pending;

      const promise = (async () => {
        try {
          const response = await savedQueryServiceClientConnect.getSavedQuery(
            createProto(GetSavedQueryRequestSchema, { name }),
            {
              contextValues: createContextValues().set(
                silentContextKey,
                silent
              ),
            }
          );
          await hydrateRelatedResources([response]);
          setCacheEntry(response, "FULL");
          // Somebody else's saved query: resolve the caller's grant level
          // before returning, because the readable/writable predicates are
          // synchronous and run as soon as this resolves. Own saved queries
          // need no policy read -- the creator short-circuit covers them.
          const me = get().currentUser?.email;
          if (me && response.creator !== `users/${me}`) {
            await get().fetchSavedQueryLevel(response);
          }
          return response;
        } catch {
          return undefined;
        } finally {
          set((s) => {
            const { [uid]: _removed, ...savedQueryRequests } =
              s.savedQueryRequests;
            return { savedQueryRequests };
          });
        }
      })();

      set((s) => ({
        savedQueryRequests: { ...s.savedQueryRequests, [uid]: promise },
      }));
      return promise;
    },

    fetchSavedQueryList: async (parent, filter, params = {}) => {
      const response = await savedQueryServiceClientConnect.searchSavedQueries(
        createProto(SearchSavedQueriesRequestSchema, {
          parent,
          filter,
          pageSize: params.pageSize,
          pageToken: params.pageToken,
        })
      );
      await hydrateRelatedResources(response.savedQueries);
      for (const savedQuery of response.savedQueries) {
        setCacheEntry(savedQuery, "BASIC");
      }
      return {
        savedQueries: response.savedQueries,
        nextPageToken: response.nextPageToken,
      };
    },

    searchSavedQueryFolders: async (parent, filter) => {
      const response =
        await savedQueryServiceClientConnect.searchSavedQueryFolders(
          createProto(SearchSavedQueryFoldersRequestSchema, {
            parent,
            filter,
          })
        );
      return response.folders;
    },

    createSavedQuery: async (savedQuery) => {
      const fullSavedQuery = savedQuery.name
        ? savedQuery
        : clone(SavedQuerySchema, savedQuery);
      const response = await savedQueryServiceClientConnect.createSavedQuery(
        createProto(CreateSavedQueryRequestSchema, {
          parent: fullSavedQuery.project,
          savedQuery: fullSavedQuery,
        })
      );
      setCacheEntry(response, "FULL");
      return response;
    },

    patchSavedQuery: async (savedQuery, updateMask, signal) => {
      if (!savedQuery.name) return undefined;
      const response = await savedQueryServiceClientConnect.updateSavedQuery(
        createProto(UpdateSavedQueryRequestSchema, {
          savedQuery,
          updateMask: { paths: updateMask },
        }),
        { signal }
      );
      setCacheEntry(response, "FULL");
      return response;
    },

    deleteSavedQueryByName: async (name) => {
      await savedQueryServiceClientConnect.deleteSavedQuery(
        createProto(DeleteSavedQueryRequestSchema, { name })
      );
      const uid = extractSavedQueryID(name);
      set((s) => {
        const {
          [cacheKey(uid, "FULL")]: _f,
          [cacheKey(uid, "BASIC")]: _b,
          ...savedQueriesByKey
        } = s.savedQueriesByKey;
        return { savedQueriesByKey };
      });
    },

    updateSavedQueryStar: async (name, starred) => {
      const response =
        await savedQueryServiceClientConnect.updateSavedQueryStar(
          createProto(UpdateSavedQueryStarRequestSchema, { name, starred })
        );
      // Refresh whichever cache views hold the row so star badges update.
      for (const view of ["FULL", "BASIC"] as const) {
        const existing = get().getSavedQueryByName(name, view);
        if (!existing) continue;
        const updated = clone(SavedQuerySchema, existing);
        updated.starred = response.starred;
        setCacheEntry(updated, view);
      }
    },

    moveMySavedQueries: async (parent, params) => {
      const response = await savedQueryServiceClientConnect.moveMySavedQueries(
        createProto(MoveMySavedQueriesRequestSchema, {
          parent,
          names: params.names,
          sourceFolder: params.sourceFolder,
          targetFolder: params.targetFolder,
        })
      );
      return response.movedCount;
    },

    patchSavedQueryFolderInCache: (names, folder) => {
      for (const name of names) {
        for (const view of ["FULL", "BASIC"] as const) {
          const existing = get().getSavedQueryByName(name, view);
          if (!existing) continue;
          const updated = clone(SavedQuerySchema, existing);
          updated.folder = folder;
          setCacheEntry(updated, view);
        }
      }
    },

    // The deduped full list. Callers split into "my" / "shared" using
    // their own current-user source — the SQL editor uses the Pinia
    // current user (reliably loaded before saved queries are fetched),
    // whereas the app store's `currentUser` can lag on routes that don't
    // load it eagerly.
    savedQueryList: () =>
      uniqBy(Object.values(get().savedQueriesByKey), (w) => w.name),

    getSavedQueryPolicy: async (name) =>
      await savedQueryServiceClientConnect.getSavedQueryPolicy(
        createProto(GetSavedQueryPolicyRequestSchema, { resource: name })
      ),

    setSavedQueryPolicy: async (name, policy) => {
      const updated = await savedQueryServiceClientConnect.setSavedQueryPolicy(
        createProto(SetSavedQueryPolicyRequestSchema, {
          resource: name,
          policy,
        })
      );
      // The caller's own level can change with the policy they just wrote --
      // an admin can demote themselves -- so drop the cached value rather than
      // leaving a stale one behind the read-only affordances.
      set((s) => {
        const { [name]: _dropped, ...savedQueryLevelByName } =
          s.savedQueryLevelByName;
        return { savedQueryLevelByName };
      });
      return updated;
    },

    fetchSavedQueryLevel: async (savedQuery) => {
      const principals = new Set(callerPrincipals());
      let level = SavedQueryBinding_Level.LEVEL_UNSPECIFIED;
      try {
        const policy = await get().getSavedQueryPolicy(savedQuery.name);
        for (const binding of policy.bindings) {
          if (binding.level <= level) continue;
          if (binding.members.some((member) => principals.has(member))) {
            level = binding.level;
          }
        }
      } catch {
        // Unreadable or unreachable: leave the level unspecified, which the
        // predicates treat as "no grant" and fall back to creator/admin.
      }
      set((s) => ({
        savedQueryLevelByName: {
          ...s.savedQueryLevelByName,
          [savedQuery.name]: level,
        },
      }));
      return level;
    },
  };
};
