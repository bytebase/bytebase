import { getCurrentUserV1 } from "@/stores";
import { getProjectByName } from "@/stores/app/projectAccess";
import { extractUserEmail } from "@/stores/modules/v1/common";
import { UNKNOWN_ID, UNKNOWN_PROJECT_NAME } from "@/types";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import { SavedQuery_Visibility } from "@/types/proto-es/v1/saved_query_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

export const extractSavedQueryID = (name: string) => {
  const pattern = /(?:^|\/)savedQueries\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// readable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and all members in the project.
export const isSavedQueryReadableV1 = (sheet: SavedQuery) => {
  const currentUser = getCurrentUserV1();

  if (extractUserEmail(sheet.creator) === currentUser.email) {
    // Always readable to the creator
    return true;
  }

  if (hasWorkspacePermissionV2("bb.worksheets.manage")) {
    return true;
  }

  switch (sheet.visibility) {
    case SavedQuery_Visibility.PRIVATE:
      return false;
    case SavedQuery_Visibility.PROJECT_READ:
    case SavedQuery_Visibility.PROJECT_WRITE: {
      const projectV1 = getProjectByName(sheet.project);
      if (projectV1.name === UNKNOWN_PROJECT_NAME) {
        return false;
      }
      return hasProjectPermissionV2(projectV1, "bb.worksheets.get");
    }
  }
  return false;
};

// writable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and project owner.
export const isSavedQueryWritableV1 = (sheet: SavedQuery) => {
  const currentUser = getCurrentUserV1();

  if (extractUserEmail(sheet.creator) === currentUser.email) {
    // Always writable to the creator
    return true;
  }

  if (hasWorkspacePermissionV2("bb.worksheets.manage")) {
    return true;
  }

  const projectV1 = getProjectByName(sheet.project);
  if (projectV1.name === UNKNOWN_PROJECT_NAME) {
    return false;
  }
  switch (sheet.visibility) {
    case SavedQuery_Visibility.PRIVATE:
      return false;
    case SavedQuery_Visibility.PROJECT_WRITE:
      return hasProjectPermissionV2(projectV1, "bb.projects.get");
    case SavedQuery_Visibility.PROJECT_READ:
      return hasProjectPermissionV2(projectV1, "bb.worksheets.manage");
  }

  return false;
};

// `extractSavedQueryConnection` moved to `@/lib/sqlEditorConnection`
// so the database lookup can go through the React app store without
// dragging `@/stores/app` into the `@/utils` import graph (which
// would create a static ESM cycle).
