import axios from "axios";
import {
  empty,
  Instance,
  IssueId,
  Pipeline,
  PipelineId,
  Principal,
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageId,
  Task,
  TaskId,
  TaskRun,
  TaskState,
  TaskStatusPatch,
  unknown,
} from "../../types";

const state: () => TaskState = () => ({});

function convertTaskRun(
  taskRun: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): TaskRun {
  const payload = taskRun.attributes.payload
    ? JSON.parse(taskRun.attributes.payload as string)
    : undefined;

  return {
    ...(taskRun.attributes as Omit<TaskRun, "id" | "payload">),
    id: parseInt(taskRun.id),
    payload,
  };
}

function convertPartial(
  task: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Task, "pipeline" | "stage"> {
  const creator = task.attributes.creator as Principal;
  const updater = task.attributes.updater as Principal;
  const payload = task.attributes.payload
    ? JSON.parse(task.attributes.payload as string)
    : undefined;

  const taskRunList: TaskRun[] = [];
  const taskRunIdList = task.relationships!.taskRun
    .data as ResourceIdentifier[];
  // Needs to iterate through taskIdList to maintain the order
  for (const idItem of taskRunIdList) {
    for (const item of includedList || []) {
      if (item.type == "taskRun") {
        if (idItem.id == item.id) {
          const taskRun: TaskRun = convertTaskRun(
            item,
            includedList,
            rootGetters
          );
          taskRunList.push(taskRun);
        }
      }
    }
  }

  const instanceId = (task.relationships!.instance.data as ResourceIdentifier)
    .id;
  let instance: Instance = empty("INSTANCE") as Instance;
  instance.id = parseInt(instanceId);

  let database = undefined;
  for (const item of includedList || []) {
    if (
      item.type == "instance" &&
      (task.relationships!.instance.data as ResourceIdentifier).id == item.id
    ) {
      instance = rootGetters["instance/convert"](item, includedList);
    }
    if (
      item.type == "database" &&
      // Tasks like creating database may not have database.
      (task.relationships!.database.data as ResourceIdentifier)?.id == item.id
    ) {
      database = rootGetters["database/convert"](item, includedList);
    }
  }

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
      | "pipeline"
      | "stage"
    >),
    id: parseInt(task.id),
    creator,
    updater,
    payload,
    instance,
    database,
    taskRunList,
  };
}

const getters = {
  convertPartial:
    (state: TaskState, getters: any, rootState: any, rootGetters: any) =>
    (task: ResourceObject, includedList: ResourceObject[]): Task => {
      // It's only called when pipeline/stage tries to convert itself, so we don't have a issue yet.
      const pipelineId = task.attributes.pipelineId as PipelineId;
      let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
      pipeline.id = pipelineId;

      const stageId = task.attributes.stageId as StageId;
      let stage: Stage = unknown("STAGE") as Stage;
      stage.id = stageId;

      return {
        ...convertPartial(task, includedList, rootGetters),
        pipeline,
        stage,
      };
    },
};

const actions = {
  async updateStatus(
    { dispatch, rootGetters }: any,
    {
      issueId,
      pipelineId,
      taskId,
      taskStatusPatch,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      taskId: TaskId;
      taskStatusPatch: TaskStatusPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/pipeline/${pipelineId}/task/${taskId}/status`, {
        data: {
          type: "taskStatusPatch",
          attributes: taskStatusPatch,
        },
      })
    ).data;
    const task = convertPartial(data.data, data.included, rootGetters);

    dispatch("issue/fetchIssueById", issueId, { root: true });

    return task;
  },
};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
