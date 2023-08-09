import axios from "axios";
import { defineStore } from "pinia";
import {
  empty,
  Instance,
  Issue,
  IssueId,
  Pipeline,
  PipelineId,
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageAllTaskStatusPatch,
  StageId,
  Task,
  TaskCheckRun,
  TaskId,
  TaskPatch,
  TaskProgress,
  TaskRun,
  TaskState,
  TaskStatusPatch,
  unknown,
} from "@/types";
import { useLegacyDatabaseStore } from "./database";
import { useLegacyInstanceStore } from "./instance";
import { useIssueStore } from "./issue";
import { getPrincipalFromIncludedList } from "./principal";

function convertTaskRun(
  taskRun: ResourceObject,
  includedList: ResourceObject[]
): TaskRun {
  const result = taskRun.attributes.result
    ? JSON.parse((taskRun.attributes.result as string) || "{}")
    : {};
  const payload = taskRun.attributes.payload
    ? JSON.parse((taskRun.attributes.payload as string) || "{}")
    : {};

  return {
    ...(taskRun.attributes as Omit<
      TaskRun,
      "id" | "result" | "payload" | "creator" | "updater"
    >),
    id: parseInt(taskRun.id),
    creator: getPrincipalFromIncludedList(
      taskRun.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      taskRun.relationships!.updater.data,
      includedList
    ),
    result,
    payload,
  };
}

function convertTaskCheckRun(
  taskCheckRun: ResourceObject,
  includedList: ResourceObject[]
): TaskCheckRun {
  const result = taskCheckRun.attributes.result
    ? JSON.parse((taskCheckRun.attributes.result as string) || "{}")
    : {};

  const payload = taskCheckRun.attributes.payload
    ? JSON.parse((taskCheckRun.attributes.payload as string) || "{}")
    : {};

  return {
    ...(taskCheckRun.attributes as Omit<
      TaskCheckRun,
      "id" | "result" | "payload" | "creator" | "updater"
    >),
    id: parseInt(taskCheckRun.id),
    creator: getPrincipalFromIncludedList(
      taskCheckRun.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      taskCheckRun.relationships!.updater.data,
      includedList
    ),
    result,
    payload,
  };
}

function convertTaskProgress(attributes: any): TaskProgress {
  if (!attributes) return unknown("TASK_PROGRESS");

  const progress: TaskProgress = { ...attributes };
  if (typeof attributes.comment === "string") {
    try {
      progress.payload = JSON.parse(attributes.comment);
    } catch {
      progress.payload = undefined;
    }
  } else {
    progress.payload = undefined;
  }

  return progress;
}

function convertPartial(
  task: ResourceObject,
  includedList: ResourceObject[]
): Omit<Task, "pipeline" | "stage"> {
  const payload = task.attributes.payload
    ? JSON.parse((task.attributes.payload as string) || "{}")
    : {};

  const taskRunList: TaskRun[] = [];
  const taskRunIdList = task.relationships!.taskRun
    .data as ResourceIdentifier[];
  // Needs to iterate through taskIdList to maintain the order
  for (const idItem of taskRunIdList) {
    for (const item of includedList || []) {
      if (item.type == "taskRun") {
        if (idItem.id == item.id) {
          const taskRun: TaskRun = convertTaskRun(item, includedList);
          taskRunList.push(taskRun);
        }
      }
    }
  }

  const taskCheckRunList: TaskCheckRun[] = [];
  const taskCheckRunIdList = task.relationships!.taskCheckRun
    .data as ResourceIdentifier[];
  // Needs to iterate through taskIdList to maintain the order
  for (const idItem of taskCheckRunIdList) {
    for (const item of includedList || []) {
      if (item.type == "taskCheckRun") {
        if (idItem.id == item.id) {
          const taskCheckRun: TaskCheckRun = convertTaskCheckRun(
            item,
            includedList
          );
          taskCheckRunList.push(taskCheckRun);
        }
      }
    }
  }

  let instance: Instance = empty("INSTANCE") as Instance;
  if (task.relationships?.instance.data) {
    const instanceId = (task.relationships.instance.data as ResourceIdentifier)
      .id;
    instance.id = parseInt(instanceId, 10);
  }

  let database = undefined;
  const databaseStore = useLegacyDatabaseStore();
  const instanceStore = useLegacyInstanceStore();
  for (const item of includedList || []) {
    if (
      item.type == "instance" &&
      (task.relationships!.instance.data as ResourceIdentifier).id == item.id
    ) {
      instance = instanceStore.convert(item, includedList);
    }
    if (
      item.type == "database" &&
      // Tasks such as creating database may not have database.
      (task.relationships!.database.data as ResourceIdentifier)?.id == item.id
    ) {
      database = databaseStore.convert(item, includedList);
    }
  }
  const progress = convertTaskProgress(task.attributes.progress);

  return {
    ...(task.attributes as Omit<
      Task,
      | "id"
      | "creator"
      | "updater"
      | "payload"
      | "instance"
      | "database"
      | "taskRunList"
      | "taskCheckRunList"
      | "pipeline"
      | "stage"
      | "progress"
    >),
    id: parseInt(task.id),
    creator: getPrincipalFromIncludedList(
      task.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      task.relationships!.updater.data,
      includedList
    ),
    payload,
    instance,
    database,
    progress,
    taskRunList,
    taskCheckRunList,
  };
}

export const useTaskStore = defineStore("task", {
  state: (): TaskState => ({}),
  actions: {
    convertPartial(task: ResourceObject, includedList: ResourceObject[]) {
      // It's only called when pipeline/stage tries to convert itself, so we don't have a issue yet.
      const pipelineId = task.attributes.pipelineId as PipelineId;
      const pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
      pipeline.id = pipelineId;

      const stageId = task.attributes.stageId as StageId;
      const stage: Stage = unknown("STAGE") as Stage;
      stage.id = stageId;

      return {
        ...convertPartial(task, includedList),
        pipeline,
        stage,
      } as Task;
    },
    async updateStatus({
      issueId,
      pipelineId,
      taskId,
      taskStatusPatch,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      taskId: TaskId;
      taskStatusPatch: TaskStatusPatch;
    }) {
      const data = (
        await axios.patch(`/api/pipeline/${pipelineId}/task/${taskId}/status`, {
          data: {
            type: "taskStatusPatch",
            attributes: taskStatusPatch,
          },
        })
      ).data;
      const task = this.convertPartial(data.data, data.included);

      useIssueStore().fetchIssueById(issueId);

      return task;
    },
    async updateStageAllTaskStatus({
      issue,
      stage,
      patch,
    }: {
      issue: Issue;
      stage: Stage;
      patch: StageAllTaskStatusPatch;
    }) {
      const { pipeline } = stage;
      await axios.patch(
        `/api/pipeline/${pipeline.id}/stage/${stage.id}/status`,
        {
          data: {
            type: "stageAllTaskStatusPatch",
            attributes: patch,
          },
        }
      );

      useIssueStore().fetchIssueById(issue.id);
    },
    async patchTask({
      issueId,
      pipelineId,
      taskId,
      taskPatch,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      taskId: TaskId;
      taskPatch: TaskPatch;
    }) {
      const data = (
        await axios.patch(`/api/pipeline/${pipelineId}/task/${taskId}`, {
          data: {
            type: "taskPatch",
            attributes: taskPatch,
          },
        })
      ).data;
      const task = this.convertPartial(data.data, data.included);

      useIssueStore().fetchIssueById(issueId);

      return task;
    },
    async patchAllTasksInIssue({
      issueId,
      pipelineId,
      taskPatch,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      taskPatch: TaskPatch;
    }) {
      await axios.patch(`/api/pipeline/${pipelineId}/task/all`, {
        data: {
          type: "taskPatch",
          attributes: taskPatch,
        },
      });

      useIssueStore().fetchIssueById(issueId);
    },
    async runChecks({
      issueId,
      pipelineId,
      taskId,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      taskId: TaskId;
    }) {
      const data = (
        await axios.post(`/api/pipeline/${pipelineId}/task/${taskId}/check`)
      ).data;
      const task = this.convertPartial(data.data, data.included);

      useIssueStore().fetchIssueById(issueId);

      return task;
    },
  },
});
