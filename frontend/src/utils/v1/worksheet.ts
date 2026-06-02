import { getProjectByName } from "@/react/stores/app/projectAccess";
import { getCurrentUserV1 } from "@/store";
import { extractUserEmail } from "@/store/modules/v1/common";
import { UNKNOWN_ID, UNKNOWN_PROJECT_NAME } from "@/types";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

export const extractWorksheetID = (name: string) => {
  const pattern = /(?:^|\/)worksheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? `${UNKNOWN_ID}`;
};

// readable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and all members in the project.
export const isWorksheetReadableV1 = (sheet: Worksheet) => {
  const currentUser = getCurrentUserV1();

  if (extractUserEmail(sheet.creator) === currentUser.email) {
    // Always readable to the creator
    return true;
  }

  if (hasWorkspacePermissionV2("bb.worksheets.manage")) {
    return true;
  }

  switch (sheet.visibility) {
    case Worksheet_Visibility.PRIVATE:
      return false;
    case Worksheet_Visibility.PROJECT_READ:
    case Worksheet_Visibility.PROJECT_WRITE: {
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
export const isWorksheetWritableV1 = (sheet: Worksheet) => {
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
    case Worksheet_Visibility.PRIVATE:
      return false;
    case Worksheet_Visibility.PROJECT_WRITE:
      return hasProjectPermissionV2(projectV1, "bb.projects.get");
    case Worksheet_Visibility.PROJECT_READ:
      return hasProjectPermissionV2(projectV1, "bb.worksheets.manage");
  }

  return false;
};

// `extractWorksheetConnection` moved to `@/react/lib/sqlEditorConnection`
// so the database lookup can go through the React app store without
// dragging `@/react/stores/app` into the `@/utils` import graph (which
// would create a static ESM cycle).
