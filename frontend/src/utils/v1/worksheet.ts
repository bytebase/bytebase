import { useAuthStore, useProjectV1Store } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import { PresetRoleType } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import { hasWorkspaceLevelRole, hasProjectPermissionV2 } from "@/utils";

export const extractWorksheetUID = (name: string) => {
  const pattern = /(?:^|\/)worksheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

// readable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and all members in the project.
export const isWorksheetReadableV1 = (sheet: Worksheet) => {
  const authStore = useAuthStore();

  if (
    getUserEmailFromIdentifier(sheet.creator) === authStore.currentUser.email
  ) {
    // Always readable to the creator
    return true;
  }

  if (
    hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_ADMIN) ||
    hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_DBA)
  ) {
    return true;
  }

  switch (sheet.visibility) {
    case Worksheet_Visibility.VISIBILITY_PRIVATE:
      return false;
    case Worksheet_Visibility.VISIBILITY_PROJECT_READ:
    case Worksheet_Visibility.VISIBILITY_PROJECT_WRITE: {
      const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
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
  const authStore = useAuthStore();

  if (
    getUserEmailFromIdentifier(sheet.creator) === authStore.currentUser.email
  ) {
    // Always writable to the creator
    return true;
  }
  if (
    hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_ADMIN) ||
    hasWorkspaceLevelRole(PresetRoleType.WORKSPACE_DBA)
  ) {
    return true;
  }

  const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
  switch (sheet.visibility) {
    case Worksheet_Visibility.VISIBILITY_PRIVATE:
      return false;
    case Worksheet_Visibility.VISIBILITY_PROJECT_WRITE:
    case Worksheet_Visibility.VISIBILITY_PROJECT_READ:
      return hasProjectPermissionV2(projectV1, "bb.worksheets.manage");
  }

  return false;
};
