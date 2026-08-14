import { getCurrentUserV1 } from "@/stores";
import { getProjectByName } from "@/stores/app/projectAccess";
import { getSavedQueryLevel } from "@/stores/app/savedQueryAccess";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  type SavedQuery,
  SavedQueryBinding_Level,
} from "@/types/proto-es/v1/saved_query_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

export const extractSavedQueryID = (name: string) => {
  const pattern = /(?:^|\/)savedQueries\/([^/]+)(?:$|\/)/;
  const matches = pattern.exec(name);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// Browsing saved queries takes bb.savedQueries.search on the project, which is
// the server's whole discovery gate -- manage does not substitute for it, since
// the search family is caller-scoped and an admin reads everyone's saved
// queries through ListSavedQueries instead. A SQL role can grant query access
// without search, in which case the tree stays empty rather than firing
// requests that can only come back denied.
export const canSearchSavedQueriesInProject = (project: string) =>
  hasProjectPermissionV2(getProjectByName(project), "bb.savedQueries.search");

// Creating a saved query takes bb.savedQueries.create on the project. A role
// can grant SQL Editor access without it, so entry points that would persist a
// new saved query check this first -- the editor stays usable, it just keeps
// the work local.
export const canCreateSavedQueryInProject = (project: string) =>
  hasProjectPermissionV2(getProjectByName(project), "bb.savedQueries.create");

// Three things grant access to a saved query, mirroring the server: being its
// creator, holding a VIEWER or EDITOR binding on it, and holding
// "bb.savedQueries.manage" (the admin backstop).
//
// The backstop is checked on the saved query's own project, mirroring the
// server: a project-level grant (Project Owner, or a custom project role)
// is enough, and the project check already falls back to workspace grants.
//
// The grant level is not on the resource -- nothing caller-relative is -- so it
// comes from the policy, resolved and cached by the app store when a saved
// query somebody else created is fetched. An unresolved level reads as "no
// grant", which is why the store awaits that resolution before handing the
// saved query to callers of these predicates.
const isCreatorOrAdmin = (sheet: SavedQuery) => {
  const currentUser = getCurrentUserV1();
  if (extractUserEmail(sheet.creator) === currentUser.email) {
    return true;
  }
  return hasProjectPermissionV2(
    getProjectByName(sheet.project),
    "bb.savedQueries.manage"
  );
};

const grantedLevel = (sheet: SavedQuery) => getSavedQueryLevel(sheet.name);

export const isSavedQueryReadableV1 = (sheet: SavedQuery) =>
  isCreatorOrAdmin(sheet) ||
  grantedLevel(sheet) >= SavedQueryBinding_Level.VIEWER;

export const isSavedQueryWritableV1 = (sheet: SavedQuery) =>
  isCreatorOrAdmin(sheet) ||
  grantedLevel(sheet) >= SavedQueryBinding_Level.EDITOR;

// Sharing, deleting, and re-filing are creator-or-admin: a binding never
// confers them, so an EDITOR grantee must not see those affordances.
export const isSavedQueryManageableV1 = (sheet: SavedQuery) =>
  isCreatorOrAdmin(sheet);

// `extractSavedQueryConnection` moved to `@/lib/sqlEditorConnection`
// so the database lookup can go through the React app store without
// dragging `@/stores/app` into the `@/utils` import graph (which
// would create a static ESM cycle).
