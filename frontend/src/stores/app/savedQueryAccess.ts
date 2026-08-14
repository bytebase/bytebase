import { SavedQueryBinding_Level } from "@/types/proto-es/v1/saved_query_service_pb";

type SavedQueryAccess = {
  getSavedQueryLevel: (name: string) => SavedQueryBinding_Level;
};

// Same indirection as `projectAccess`: the readable/writable predicates live in
// `@/utils`, which sits inside the app store's own load graph, so they cannot
// statically import the store index without forming an initialization cycle.
// The slice registers the real implementation via `setSavedQueryAccess` at
// store-creation time.
//
// Before registration, and for any saved query whose policy has not been read,
// this reports no grant — the predicates then fall back to creator/admin, which
// is the safe direction: affordances stay hidden rather than appearing for
// someone the server will refuse.
let savedQueryAccess: SavedQueryAccess = {
  getSavedQueryLevel: () => SavedQueryBinding_Level.LEVEL_UNSPECIFIED,
};

export const setSavedQueryAccess = (access: SavedQueryAccess) => {
  savedQueryAccess = access;
};

export const getSavedQueryLevel = (name: string) =>
  savedQueryAccess.getSavedQueryLevel(name);
