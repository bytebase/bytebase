import { uniq } from "lodash-es";
import { flattenTaskList } from "@/components/Issue/logic";
import { useSheetV1Store } from "@/store";
import { getSheetPathByLegacyProject } from "@/store/modules/v1/common";
import {
  Issue,
  SheetId,
  SheetIssueBacktracePayload,
  Task,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseSchemaUpdateSDLPayload,
  UNKNOWN_ID,
} from "@/types";

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
    if (sheetId && sheetId !== UNKNOWN_ID) {
      sheetIdList.push(sheetId);
    }
  });

  const sheetV1Store = useSheetV1Store();
  const requests = uniq(sheetIdList).map((sheetId) => {
    const payload: SheetIssueBacktracePayload = {
      type: "bb.sheet.issue-backtrace",
      issueId: issue.id,
      issueName: issue.name,
    };

    return sheetV1Store.patchSheet({
      name: getSheetPathByLegacyProject(issue.project, sheetId),
      payload: JSON.stringify(payload),
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
