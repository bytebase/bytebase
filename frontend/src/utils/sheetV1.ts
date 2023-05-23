import { isUndefined } from "lodash-es";

import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { SheetIssueBacktracePayload } from "@/types";
import {
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
  isMemberOfProjectV1,
} from "../utils";
import { Sheet, Sheet_Visibility } from "@/types/proto/v1/sheet_service";
import {
  getUserEmailFromIdentifier,
  getProjectAndSheetId,
} from "@/store/modules/v1/common";

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

    const projectV1 = useProjectV1Store().getProjectByName(projectId);
    return isMemberOfProjectV1(projectV1.iamPolicy, currentUserV1.value);
  }
  // visibility === "PUBLIC"
  return true;
};

export const isSheetWritableV1 = (sheet: Sheet) => {
  // If the sheet is linked to an issue, it's NOT writable
  if (getSheetIssueBacktracePayloadV1(sheet)) {
    return false;
  }

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
    const projectV1 = useProjectV1Store().getProjectByName(projectId);
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

export const getSheetIssueBacktracePayloadV1 = (sheet: Sheet) => {
  const maybePayload = JSON.parse(
    sheet.payload ?? "{}"
  ) as SheetIssueBacktracePayload;
  if (
    maybePayload.type === "bb.sheet.issue-backtrace" &&
    !isUndefined(maybePayload.issueId) &&
    !isUndefined(maybePayload.issueName)
  ) {
    return maybePayload;
  }

  return undefined;
};
