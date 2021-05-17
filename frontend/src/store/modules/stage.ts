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
  Principal,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "pipeline"> {
  const creator = stage.attributes.creator as Principal;
  const updater = stage.attributes.updater as Principal;

  const environmentId = (
    stage.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentId);

  const taskList: Task[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (stage.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = rootGetters["environment/convert"](item, includedList);
    }

    if (item.type == "task") {
      const taskIdList = stage.relationships!.task.data as ResourceIdentifier[];
      for (const idItem of taskIdList) {
        if (idItem.id == item.id) {
          const task: Task = rootGetters["task/convertPartial"](
            item,
            includedList
          );
          taskList.push(task);
        }
      }
    }
  }

  const result: Omit<Stage, "pipeline"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "creator" | "updater" | "database" | "taskList"
    >),
    id: parseInt(stage.id),
    creator,
    updater,
    environment,
    taskList,
  };

  return result;
}

const getters = {
  convertPartial:
    (state: StageState, getters: any, rootState: any, rootGetters: any) =>
    (stage: ResourceObject, includedList: ResourceObject[]): Stage => {
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
