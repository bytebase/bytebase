import Long from "long";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import {
  getUserEmailFromIdentifier,
  getProjectAndSheetId,
} from "@/store/modules/v1/common";
import { Sheet, Sheet_Visibility } from "@/types/proto/v1/sheet_service";
import {
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
  isMemberOfProjectV1,
} from "../../utils";

export const extractSheetUID = (name: string) => {
  const pattern = /(?:^|\/)sheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

export const isLocalSheet = (name: string) => {
  return extractSheetUID(name).startsWith("-");
};

export const isSheetReadableV1 = (sheet: Sheet) => {
  const currentUserV1 = useCurrentUserV1();

  // readable to
  // PRIVATE: the creator only
  // PROJECT: the creator and members in the project, workspace Owner and DBA
  // PUBLIC: everyone

  if (getUserEmailFromIdentifier(sheet.creator) === currentUserV1.value.email) {
    // Always readable to the creator
    return true;
  }
  const { visibility } = sheet;
  if (visibility === Sheet_Visibility.VISIBILITY_PRIVATE) {
    return false;
  }
  if (visibility === Sheet_Visibility.VISIBILITY_PROJECT) {
    if (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-project",
        currentUserV1.value.userRole
      )
    ) {
      return true;
    }

    const [projectId, _] = getProjectAndSheetId(sheet.name);

    const projectV1 = useProjectV1Store().getProjectByName(
      `projects/${projectId}`
    );
    return isMemberOfProjectV1(projectV1.iamPolicy, currentUserV1.value);
  }
  // visibility === "PUBLIC"
  return true;
};

export const isSheetWritableV1 = (sheet: Sheet) => {
  const currentUserV1 = useCurrentUserV1();

  // writable to
  // PRIVATE: the creator only
  // PROJECT: the creator or project role can manage sheet, workspace Owner and DBA
  // PUBLIC: the creator only

  if (getUserEmailFromIdentifier(sheet.creator) === currentUserV1.value.email) {
    // Always writable to the creator
    return true;
  }
  const { visibility } = sheet;
  if (visibility === Sheet_Visibility.VISIBILITY_PRIVATE) {
    return false;
  }
  if (visibility === Sheet_Visibility.VISIBILITY_PROJECT) {
    if (
      hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-project",
        currentUserV1.value.userRole
      )
    ) {
      return true;
    }

    const [projectId, _] = getProjectAndSheetId(sheet.name);
    const projectV1 = useProjectV1Store().getProjectByName(
      `projects/${projectId}`
    );
    const isCurrentUserProjectOwner = () => {
      return hasPermissionInProjectV1(
        projectV1.iamPolicy,
        currentUserV1.value,
        "bb.permission.project.manage-sheet"
      );
    };
    return isCurrentUserProjectOwner();
  }
  // visibility === "PUBLIC"
  return false;
};

export const setSheetStatement = (sheet: Sheet, statement: string) => {
  sheet.content = new TextEncoder().encode(statement);
  sheet.contentSize = Long.fromNumber(statement.length);
};

export const getSheetStatement = (sheet: Sheet) => {
  return new TextDecoder().decode(sheet.content);
};
