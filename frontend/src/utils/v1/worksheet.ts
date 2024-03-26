import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import { PresetRoleType } from "@/types";
import type { Worksheet } from "@/types/proto/v1/worksheet_service";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import { isMemberOfProjectV1, isOwnerOfProjectV1 } from "@/utils";

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
  const currentUserV1 = useCurrentUserV1();

  if (getUserEmailFromIdentifier(sheet.creator) === currentUserV1.value.email) {
    // Always readable to the creator
    return true;
  }

  if (
    currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN) ||
    currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_DBA)
  ) {
    return true;
  }

  switch (sheet.visibility) {
    case Worksheet_Visibility.VISIBILITY_PRIVATE:
      return false;
    case Worksheet_Visibility.VISIBILITY_PROJECT_READ:
    case Worksheet_Visibility.VISIBILITY_PROJECT_WRITE: {
      const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
      return isMemberOfProjectV1(projectV1.iamPolicy, currentUserV1.value);
    }
  }
  return false;
};

// writable to
// PRIVATE: workspace Owner/DBA and the creator only.
// PROJECT_WRITE: workspace Owner/DBA and all members in the project.
// PROJECT_READ: workspace Owner/DBA and project owner.
export const isWorksheetWritableV1 = (sheet: Worksheet) => {
  const currentUserV1 = useCurrentUserV1();

  if (getUserEmailFromIdentifier(sheet.creator) === currentUserV1.value.email) {
    // Always writable to the creator
    return true;
  }
  if (
    currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN) ||
    currentUserV1.value.roles.includes(PresetRoleType.WORKSPACE_DBA)
  ) {
    return true;
  }

  const projectV1 = useProjectV1Store().getProjectByName(sheet.project);
  switch (sheet.visibility) {
    case Worksheet_Visibility.VISIBILITY_PRIVATE:
      return false;
    case Worksheet_Visibility.VISIBILITY_PROJECT_WRITE:
      return isMemberOfProjectV1(projectV1.iamPolicy, currentUserV1.value);
    case Worksheet_Visibility.VISIBILITY_PROJECT_READ:
      return isOwnerOfProjectV1(projectV1.iamPolicy, currentUserV1.value);
  }

  return false;
};
