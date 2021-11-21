
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageState,
  Task,
  unknown,
  PipelineID,
  Pipeline,
  Environment,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "pipeline"> {
  const environmentID = (
    stage.relationships!.environment.data as ResourceIdentifier
  ).id;
  let environment: Environment = unknown("ENVIRONMENT") as Environment;
  environment.id = parseInt(environmentID);

  const taskList: Task[] = [];
  const taskIDList = stage.relationships!.task.data as ResourceIdentifier[];
  // Needs to iterate through taskIDList to maintain the order
  for (const idItem of taskIDList) {
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
    ...(stage.attributes as Omit<Stage, "id" | "database" | "taskList">),
    id: parseInt(stage.id),
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
      const pipelineID = stage.attributes.pipelineID as PipelineID;
      let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
      pipeline.id = pipelineID;

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
