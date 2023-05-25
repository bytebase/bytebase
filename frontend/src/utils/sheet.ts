import { isUndefined, uniq } from "lodash-es";

import { useSheetStore } from "@/store";
import {
  Issue,
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
import { flattenTaskList } from "@/components/Issue/logic";

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
