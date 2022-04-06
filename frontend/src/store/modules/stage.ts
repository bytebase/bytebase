import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageState,
  Task,
  unknown,
  PipelineId,
  Pipeline,
  Environment,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "pipeline"> {
  let environment = unknown("ENVIRONMENT") as Environment;
  if (stage.relationships?.environment.data) {
    const environmentId = (
      stage.relationships.environment.data as ResourceIdentifier
    ).id;
    environment.id = parseInt(environmentId, 10);
  }

  const taskList: Task[] = [];
  const taskIdList = stage.relationships!.task.data as ResourceIdentifier[];
  // Needs to iterate through taskIdList to maintain the order
  for (const idItem of taskIdList) {
    for (const item of includedList || []) {
      if (item.type == "task") {
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

  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (stage.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = rootGetters["environment/convert"](item, includedList);
    }
  }

  const result: Omit<Stage, "pipeline"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "database" | "taskList" | "creator" | "updater"
    >),
    id: parseInt(stage.id),
    creator: getPrincipalFromIncludedList(
      stage.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      stage.relationships!.updater.data,
      includedList
    ),
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
      const pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
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
