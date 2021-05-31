import axios from "axios";
import {
  ResourceObject,
  TaskState,
  Task,
  Stage,
  unknown,
  IssueId,
  StageId,
  PipelineId,
  Pipeline,
  TaskStatusPatch,
  TaskId,
  Database,
  empty,
  ResourceIdentifier,
  Principal,
  TaskRun,
} from "../../types";

const state: () => TaskState = () => ({});

function convertTaskRun(
  taskRun: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): TaskRun {
  const creator = taskRun.attributes.creator as Principal;
  const updater = taskRun.attributes.updater as Principal;

  return {
    ...(taskRun.attributes as Omit<TaskRun, "id" | "creator" | "updater">),
    id: parseInt(taskRun.id),
    creator,
    updater,
  };
}

function convertPartial(
  task: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Task, "pipeline" | "stage"> {
  const creator = task.attributes.creator as Principal;
  const updater = task.attributes.updater as Principal;

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

  const databaseId = (task.relationships!.database.data as ResourceIdentifier)
    .id;
  let database: Database = empty("DATABASE") as Database;
  database.id = parseInt(databaseId);
  for (const item of includedList || []) {
    if (
      item.type == "database" &&
      (task.relationships!.database.data as ResourceIdentifier).id == item.id
    ) {
      database = rootGetters["database/convert"](item);
    }
  }

  return {
    ...(task.attributes as Omit<
      Task,
      | "id"
      | "creator"
      | "updater"
      | "database"
      | "taskRunList"
      | "pipeline"
      | "stage"
    >),
    id: parseInt(task.id),
    creator,
    updater,
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
    { rootGetters }: any,
    {
      pipelineId,
      taskId,
      taskStatusPatch,
    }: {
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
    return convertPartial(data.data, data.included, rootGetters);
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
