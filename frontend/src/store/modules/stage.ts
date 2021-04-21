import axios from "axios";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageId,
  StageState,
  Task,
  Issue,
  IssueId,
  unknown,
  PipelineId,
  Pipeline,
  Database,
  empty,
  Environment,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "pipeline"> {
  const creator = rootGetters["principal/principalById"](
    stage.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    stage.attributes.updaterId
  );

  // We should always have a valid environment
  const environment: Environment = rootGetters["environment/environmentById"](
    (stage.relationships!.environment.data as ResourceIdentifier).id
  );

  let database: Database = empty("DATABASE") as Database;
  // For create database stage, there is no database id.
  if (stage.relationships!.database.data) {
    database = rootGetters["database/databaseById"](
      (stage.relationships!.database.data as ResourceIdentifier).id
    );
  }

  const taskList: Task[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "task" &&
      (item.relationships!.stage.data as ResourceIdentifier).id == stage.id
    ) {
      const task = rootGetters["task/convertPartial"](item);
      taskList.push(task);
    }
  }

  const result: Omit<Stage, "pipeline"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "creator" | "updater" | "database" | "taskList"
    >),
    id: stage.id,
    creator,
    updater,
    environment,
    database,
    taskList,
  };

  return result;
}

const getters = {
  convertPartial: (
    state: StageState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (stage: ResourceObject, includedList: ResourceObject[]): Stage => {
    // It's only called when pipeline tries to convert itself, so we don't have a issue yet.
    const pipelineId = stage.attributes.pipelineId as PipelineId;
    let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
    pipeline.id = pipelineId;

    const result: Stage = {
      ...convertPartial(stage, includedList, rootGetters),
      pipeline,
    };

    for (const task of result.taskList) {
      task.stage = result;
      task.pipeline = pipeline;
    }

    return result;
  },
};

const actions = {};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
