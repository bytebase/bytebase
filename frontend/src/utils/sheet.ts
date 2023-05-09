import { isUndefined, uniq } from "lodash-es";

import { useSheetStore } from "@/store";
import {
  Issue,
  Principal,
  Sheet,
  SheetId,
  SheetIssueBacktracePayload,
  SheetPayload,
  SheetSource,
  Task,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseSchemaUpdateSDLPayload,
} from "@/types";
import {
  hasPermissionInProject,
  hasWorkspacePermission,
  isMemberOfProject,
} from "../utils";
import { flattenTaskList } from "@/components/Issue/logic";

export const isSheetReadable = (sheet: Sheet, currentUser: Principal) => {
  // readable to
  // PRIVATE: the creator only
  // PROJECT: the creator and members in the project, workspace Owner and DBA
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
    if (
      hasWorkspacePermission(
        "bb.permission.workspace.manage-project",
        currentUser.role
      )
    ) {
      return true;
    }

    return isMemberOfProject(sheet.project, currentUser);
  }
  // visibility === "PUBLIC"
  return true;
};

export const isSheetWritable = (sheet: Sheet, currentUser: Principal) => {
  // If the sheet is linked to an issue, it's NOT writable
  if (getSheetIssueBacktracePayload(sheet)) {
    return false;
  }

  // writable to
  // PRIVATE: the creator only
  // PROJECT: the creator or project role can manage sheet, workspace Owner and DBA
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
    if (
      hasWorkspacePermission(
        "bb.permission.workspace.manage-project",
        currentUser.role
      )
    ) {
      return true;
    }

    const isCurrentUserProjectOwner = () => {
      return hasPermissionInProject(
        sheet.project,
        currentUser,
        "bb.permission.project.manage-sheet"
      );
    };
    return isCurrentUserProjectOwner();
  }
  // visibility === "PUBLIC"
  return false;
};

// getDefaultSheetPayloadWithSource gets the default payload with sheet source.
export const getDefaultSheetPayloadWithSource = (
  sheetSource: SheetSource
): SheetPayload => {
  if (sheetSource === "BYTEBASE") {
    // As we don't save any data for sheet from UI, return an empty payload.
    return {};
  }

  // Shouldn't reach this line.
  // For those sheet from VCS, we create and patch them in backend.
  return {};
};

export const sheetIdOfTask = (task: Task) => {
  switch (task.type) {
    case "bb.task.database.create":
      return (
        ((task as Task).payload as TaskDatabaseCreatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update-sdl":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateSDLPayload)
          .sheetId || undefined
      );
    case "bb.task.database.data.update":
      return (
        ((task as Task).payload as TaskDatabaseDataUpdatePayload).sheetId ||
        undefined
      );
    case "bb.task.database.schema.update.ghost.sync":
      return (
        ((task as Task).payload as TaskDatabaseSchemaUpdateGhostSyncPayload)
          .sheetId || undefined
      );
    default:
      return undefined;
  }
};

/**
 * If the created issue contains SQL files uploaded as sheets
 * we should patch the sheets' payloads with issueID and taskID
 * to make a sheet backtrace-able to the issue/task it belongs to.
 * Then we can display the backtrace issue link in the sheet list.
 */
export const maybeSetSheetBacktracePayloadByIssue = async (issue: Issue) => {
  const sheetIdList: SheetId[] = [];

  flattenTaskList(issue).forEach((task) => {
    const sheetId = sheetIdOfTask(task as Task);
    if (sheetId) {
      sheetIdList.push(sheetId);
    }
  });

  const store = useSheetStore();
  const requests = uniq(sheetIdList).map((sheetId) => {
    const payload: SheetIssueBacktracePayload = {
      type: "bb.sheet.issue-backtrace",
      issueId: issue.id,
      issueName: issue.name,
    };
    return store.patchSheetById({
      id: sheetId,
      payload,
    });
  });

  try {
    await Promise.all(requests);
  } catch {
    // nothing
  }
};

export const getBacktracePayloadWithIssue = (issue: Issue) => {
  return {
    type: "bb.sheet.issue-backtrace",
    issueId: issue.id,
    issueName: issue.name,
  };
};

export const getSheetIssueBacktracePayload = (sheet: Sheet) => {
  const maybePayload = (sheet.payload ?? {}) as SheetIssueBacktracePayload;
  if (
    maybePayload.type === "bb.sheet.issue-backtrace" &&
    !isUndefined(maybePayload.issueId) &&
    !isUndefined(maybePayload.issueName)
  ) {
    return maybePayload;
  }

  return undefined;
};
