import { useIssueStore, useSheetV1Store, useTaskStore } from "@/store";
import { ComposedIssue, TaskPatch } from "@/types";
import { Task } from "@/types/proto/v1/rollout_service";
import { extractSheetUID, setSheetStatement } from "@/utils";
import { createEmptyLocalSheet } from "./sheet";

export const patchLegacyIssueTasksStatement = async (
  issue: ComposedIssue,
  tasks: Task[],
  statement: string
) => {
  const sheetCreate = {
    ...createEmptyLocalSheet(),
    title: issue.title,
  };
  setSheetStatement(sheetCreate, statement);
  const createdSheet = await useSheetV1Store().createSheet(
    issue.project,
    sheetCreate
  );

  const legacyIssue = await useIssueStore().getOrFetchIssueById(
    Number(issue.uid)
  );
  for (let i = 0; i < tasks.length; i++) {
    const task = tasks[i];
    const taskPatch: TaskPatch = {
      sheetId: Number(extractSheetUID(createdSheet.name)),
    };
    await useTaskStore().patchTask({
      issueId: legacyIssue.id,
      pipelineId: legacyIssue.pipeline!.id,
      taskId: Number(task.uid),
      taskPatch,
    });
  }
};
