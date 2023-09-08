import { useIssueStore, useSheetV1Store, useTaskStore } from "@/store";
import { ComposedIssue, Issue as LegacyIssue, TaskPatch } from "@/types";
import { Task } from "@/types/proto/v1/rollout_service";
import { extractSheetUID, setSheetStatement } from "@/utils";
import { createEmptyLocalSheet } from "./sheet";

export const batchPatchLegacyIssueTasks = async (
  issue: ComposedIssue,
  tasks: Task[],
  patchGetter: (
    legacyIssue: LegacyIssue,
    task: Task
  ) => TaskPatch | Promise<TaskPatch>
) => {
  const legacyIssue = await useIssueStore().getOrFetchIssueById(
    Number(issue.uid)
  );
  for (let i = 0; i < tasks.length; i++) {
    const task = tasks[i];
    const taskPatch = await patchGetter(legacyIssue, task);
    await useTaskStore().patchTask({
      issueId: legacyIssue.id,
      pipelineId: legacyIssue.pipeline!.id,
      taskId: Number(task.uid),
      taskPatch,
    });
  }
};

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

  await batchPatchLegacyIssueTasks(issue, tasks, () => {
    return {
      sheetId: Number(extractSheetUID(createdSheet.name)),
    };
  });
};
