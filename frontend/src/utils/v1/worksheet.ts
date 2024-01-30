import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import { hasProjectPermissionV2 } from "@/utils";

export const extractWorksheetUID = (name: string) => {
  const pattern = /(?:^|\/)worksheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

export const isWorksheetReadableV1 = (sheet: Worksheet) => {
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
  if (visibility === Worksheet_Visibility.VISIBILITY_PRIVATE) {
    return false;
  }
  if (visibility === Worksheet_Visibility.VISIBILITY_PROJECT) {
    const projectV1 = useProjectV1Store().getProjectByName(sheet.project);

    return hasProjectPermissionV2(
      projectV1,
      currentUserV1.value,
      "bb.projects.get"
    );
  }
  // visibility === "PUBLIC"
  return true;
};

export const isWorksheetWritableV1 = (sheet: Worksheet) => {
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
  if (visibility === Worksheet_Visibility.VISIBILITY_PRIVATE) {
    return false;
  }
  if (visibility === Worksheet_Visibility.VISIBILITY_PROJECT) {
    const projectV1 = useProjectV1Store().getProjectByName(sheet.project);

    return hasProjectPermissionV2(
      projectV1,
      currentUserV1.value,
      "bb.projects.get"
    );
  }
  // visibility === "PUBLIC"
  return false;
};
