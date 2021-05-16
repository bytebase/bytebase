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
} from "../../types";

const state: () => TaskState = () => ({});

function convertPartial(
  task: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Task, "pipeline" | "stage"> {
  const creatorId = (task.relationships!.creator.data as ResourceIdentifier).id;
  let creator: Principal = unknown("PRINCIPAL") as Principal;
  creator.id = creatorId;

  const updaterId = (task.relationships!.updater.data as ResourceIdentifier).id;
  let updater: Principal = unknown("PRINCIPAL") as Principal;
  updater.id = updaterId;

  const databaseId = (task.relationships!.database.data as ResourceIdentifier)
    .id;
  let database: Database = empty("DATABASE") as Database;
  database.id = databaseId;
  for (const item of includedList || []) {
    if (
      item.type == "principal" &&
      (task.relationships!.creator.data as ResourceIdentifier).id == item.id
    ) {
      creator = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (task.relationships!.updater.data as ResourceIdentifier).id == item.id
    ) {
      updater = rootGetters["principal/convert"](item);
    }

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
      "id" | "creator" | "updater" | "database" | "pipeline" | "stage"
    >),
    id: task.id,
    creator,
    updater,
    database,
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
    { dispatch }: any,
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
    // TODO: Returns the updated pipeline and update the issue.
    const data = (
      await axios.patch(`/api/pipeline/${pipelineId}/task/${taskId}/status`, {
        data: {
          type: "taskstatuspatch",
          attributes: taskStatusPatch,
        },
      })
    ).data;

    dispatch("issue/fetchIssueById", issueId, { root: true });
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
