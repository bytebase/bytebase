import axios from "axios";
import {
  ResourceIdentifier,
  ResourceObject,
  Task,
  TaskId,
  TaskState,
  Step,
  Issue,
  IssueId,
  unknown,
  PipelineId,
  Pipeline,
  Database,
  empty,
  Environment,
} from "../../types";

const state: () => TaskState = () => ({});

function convertPartial(
  task: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Task, "pipeline"> {
  const creator = rootGetters["principal/principalById"](
    task.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    task.attributes.updaterId
  );

  // We should always have a valid environment
  const environment: Environment = rootGetters["environment/environmentById"](
    (task.relationships!.environment.data as ResourceIdentifier).id
  );

  let database: Database = empty("DATABASE") as Database;
  // For create database task, there is no database id.
  if (task.relationships!.database.data) {
    database = rootGetters["database/databaseById"](
      (task.relationships!.database.data as ResourceIdentifier).id
    );
  }

  const stepList: Step[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "step" &&
      (item.relationships!.task.data as ResourceIdentifier).id == task.id
    ) {
      const step = rootGetters["step/convertPartial"](item);
      stepList.push(step);
    }
  }

  const result: Omit<Task, "pipeline"> = {
    ...(task.attributes as Omit<
      Task,
      "id" | "creator" | "updater" | "database" | "stepList"
    >),
    id: task.id,
    creator,
    updater,
    environment,
    database,
    stepList,
  };

  return result;
}

const getters = {
  convertPartial: (
    state: TaskState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (task: ResourceObject, includedList: ResourceObject[]): Task => {
    // It's only called when pipeline tries to convert itself, so we don't have a issue yet.
    const pipelineId = task.attributes.pipelineId as PipelineId;
    let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
    pipeline.id = pipelineId;

    const result: Task = {
      ...convertPartial(task, includedList, rootGetters),
      pipeline,
    };

    for (const step of result.stepList) {
      step.task = result;
      step.pipeline = pipeline;
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
