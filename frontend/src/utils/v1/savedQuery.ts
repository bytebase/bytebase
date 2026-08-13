import { getCurrentUserV1 } from "@/stores";
import { getProjectByName } from "@/stores/app/projectAccess";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

export const extractSavedQueryID = (name: string) => {
  const pattern = /(?:^|\/)savedQueries\/([^/]+)(?:$|\/)/;
  const matches = pattern.exec(name);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// Saved queries are private: only the creator, or someone holding
// "bb.savedQueries.manage" (the admin backstop), can read or write one.
// Per-object sharing arrives with the access-model redesign.
//
// The backstop is checked on the saved query's own project, mirroring the
// server: a project-level grant (Project Owner, or a custom project role)
// is enough, and the project check already falls back to workspace grants.
const canAccessSavedQuery = (sheet: SavedQuery) => {
  const currentUser = getCurrentUserV1();
  if (extractUserEmail(sheet.creator) === currentUser.email) {
    return true;
  }
  return hasProjectPermissionV2(
    getProjectByName(sheet.project),
    "bb.savedQueries.manage"
  );
};

export const isSavedQueryReadableV1 = (sheet: SavedQuery) =>
  canAccessSavedQuery(sheet);

export const isSavedQueryWritableV1 = (sheet: SavedQuery) =>
  canAccessSavedQuery(sheet);

// `extractSavedQueryConnection` moved to `@/lib/sqlEditorConnection`
// so the database lookup can go through the React app store without
// dragging `@/stores/app` into the `@/utils` import graph (which
// would create a static ESM cycle).
