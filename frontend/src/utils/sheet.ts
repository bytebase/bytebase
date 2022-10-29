import { Principal, Sheet, SheetPayload, TabMode } from "@/types";
import { hasProjectPermission } from "../utils";

export const isSheetReadable = (sheet: Sheet, currentUser: Principal) => {
  // readable to
  // PRIVATE: the creator only
  // PROJECT: the creator and members in the project
  // PUBLIC: everyone

  if (sheet.creator.id === currentUser.id) {
    // Always readable to the creator
    return true;
  }
  const { visibility } = sheet;
  if (visibility === "PRIVATE") {
    return false;
  }
  if (visibility === "PROJECT") {
    const projectMemberList = sheet.project.memberList;
    return (
      projectMemberList.findIndex(
        (member) => member.principal.id === currentUser.id
      ) >= 0
    );
  }
  // visibility === "PUBLIC"
  return true;
};

export const isSheetWritable = (sheet: Sheet, currentUser: Principal) => {
  // writable to
  // PRIVATE: the creator only
  // PROJECT: the creator or project role can manage sheet
  // PUBLIC: the creator only

  if (sheet.creator.id === currentUser.id) {
    // Always writable to the creator
    return true;
  }
  const { visibility } = sheet;
  if (visibility === "PRIVATE") {
    return false;
  }
  if (visibility === "PROJECT") {
    const isCurrentUserProjectOwner = () => {
      const projectMemberList = sheet.project.memberList || [];
      const memberInProject = projectMemberList.find((member) => {
        return member.principal.id === currentUser.id;
      });

      return (
        memberInProject &&
        hasProjectPermission(
          "bb.permission.project.manage-sheet",
          memberInProject.role
        )
      );
    };
    return isCurrentUserProjectOwner();
  }
  // visibility === "PUBLIC"
  return false;
};

export const getDefaultSheetPayload = (): SheetPayload => {
  return {
    tabMode: TabMode.ReadOnly,
  };
};
