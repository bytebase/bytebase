import { getCurrentUserV1 } from "@/stores";
import { getProjectByName } from "@/stores/app/projectAccess";
import { getSavedQueryLevel } from "@/stores/app/savedQueryAccess";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  type SavedQuery,
  SavedQueryBinding_Level,
} from "@/types/proto-es/v1/saved_query_service_pb";
import { hasProjectWidePermissionV2 } from "@/utils";

export const extractSavedQueryID = (name: string) => {
  const pattern = /(?:^|\/)savedQueries\/([^/]+)(?:$|\/)/;
  const matches = pattern.exec(name);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// Browsing saved queries takes bb.savedQueries.search on the project, which is
// the server's whole discovery gate. A SQL role can grant query access
// without search, in which case the tree stays empty rather than firing
// requests that can only come back denied.
//
// All checks in this file are project-wide, mirroring the server: a binding
// whose condition scopes resources confers no saved-query permission.
export const canSearchSavedQueriesInProject = (project: string) =>
  hasProjectWidePermissionV2(
    getProjectByName(project),
    "bb.savedQueries.search"
  );

// Creating a saved query takes bb.savedQueries.create on the project. A role
// can grant SQL Editor access without it, so entry points that would persist a
// new saved query check this first -- the editor stays usable, it just keeps
// the work local.
export const canCreateSavedQueryInProject = (project: string) =>
  hasProjectWidePermissionV2(
    getProjectByName(project),
    "bb.savedQueries.create"
  );

// Access mirrors the server: the creator owns the saved query and holds every
// permission on it; a VIEWER binding grants bb.savedQueries.get and an EDITOR
// binding grants get + update on that saved query; the same permissions
// granted on the project reach every saved query there. The project check
// already falls back to workspace grants.
//
// The grant level is not on the resource -- nothing caller-relative is -- so it
// comes from the policy, resolved and cached by the app store when a saved
// query somebody else created is fetched. An unresolved level reads as "no
// grant", which is why the store awaits that resolution before handing the
// saved query to callers of these predicates.
const isCreator = (sheet: SavedQuery) =>
  extractUserEmail(sheet.creator) === getCurrentUserV1().email;

const hasProjectVerb = (
  sheet: SavedQuery,
  permission:
    | "bb.savedQueries.get"
    | "bb.savedQueries.update"
    | "bb.savedQueries.delete"
    | "bb.savedQueries.getIamPolicy"
    | "bb.savedQueries.setIamPolicy"
) => hasProjectWidePermissionV2(getProjectByName(sheet.project), permission);

const bindingGrantsGet = (sheet: SavedQuery) => {
  const level = getSavedQueryLevel(sheet.name);
  return (
    level === SavedQueryBinding_Level.VIEWER ||
    level === SavedQueryBinding_Level.EDITOR
  );
};

const bindingGrantsUpdate = (sheet: SavedQuery) =>
  getSavedQueryLevel(sheet.name) === SavedQueryBinding_Level.EDITOR;

export const isSavedQueryReadableV1 = (sheet: SavedQuery) =>
  isCreator(sheet) ||
  bindingGrantsGet(sheet) ||
  hasProjectVerb(sheet, "bb.savedQueries.get");

export const isSavedQueryWritableV1 = (sheet: SavedQuery) =>
  isCreator(sheet) ||
  bindingGrantsUpdate(sheet) ||
  hasProjectVerb(sheet, "bb.savedQueries.update");

// Bindings never carry deletion.
export const isSavedQueryDeletableV1 = (sheet: SavedQuery) =>
  isCreator(sheet) || hasProjectVerb(sheet, "bb.savedQueries.delete");

// Sharing: the creator, or a project-level bb.savedQueries.setIamPolicy —
// which no predefined role carries. The write is compare-and-swap over the
// policy's etag, so exercising it requires reading the policy first: the
// affordance demands getIamPolicy alongside setIamPolicy.
export const isSavedQueryShareableV1 = (sheet: SavedQuery) =>
  isCreator(sheet) ||
  (hasProjectVerb(sheet, "bb.savedQueries.setIamPolicy") &&
    hasProjectVerb(sheet, "bb.savedQueries.getIamPolicy"));

// `extractSavedQueryConnection` moved to `@/lib/sqlEditorConnection`
// so the database lookup can go through the React app store without
// dragging `@/stores/app` into the `@/utils` import graph (which
// would create a static ESM cycle).
