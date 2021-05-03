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
} from "../../types";

const state: () => TaskState = () => ({});

function convertPartial(
  task: ResourceObject,
  rootGetters: any
): Omit<Task, "pipeline" | "stage"> {
  const creator = rootGetters["principal/principalById"](
    task.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    task.attributes.updaterId
  );

  let database: Database = empty("DATABASE") as Database;
  // For create database stage, there is no database id.
  if (task.relationships!.database.data) {
    database = rootGetters["database/databaseById"](
      (task.relationships!.database.data as ResourceIdentifier).id
    );
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
  convertPartial: (
    state: TaskState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (task: ResourceObject): Task => {
    // It's only called when pipeline/stage tries to convert itself, so we don't have a issue yet.
    const pipelineId = task.attributes.pipelineId as PipelineId;
    let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
    pipeline.id = pipelineId;

    const stageId = task.attributes.stageId as StageId;
    let stage: Stage = unknown("STAGE") as Stage;
    stage.id = stageId;

    return {
      ...convertPartial(task, rootGetters),
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
      await axios.patch(`/mock/pipeline/${pipelineId}/task/${taskId}/status`, {
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
