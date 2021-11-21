
import {
  ResourceIdentifier,
  ResourceObject,
  Pipeline,
  PipelineState,
  Stage,
} from "../../types";

const state: () => PipelineState = () => ({});

function convert(
  pipeline: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Pipeline {
  const stageList: Stage[] = [];
  const stageIDList = pipeline.relationships!.stage
    .data as ResourceIdentifier[];
  // Needs to iterate through stageIDList to maintain the order
  for (const idItem of stageIDList) {
    for (const item of includedList || []) {
      if (item.type == "stage") {
        if (idItem.id == item.id) {
          const stage: Stage = rootGetters["stage/convertPartial"](
            item,
            includedList
          );
          stageList.push(stage);
        }
      }
    }
  }

  const result: Pipeline = {
    ...(pipeline.attributes as Omit<Pipeline, "id" | "stageList">),
    id: parseInt(pipeline.id),
    stageList,
  };

  // Now we have a complate issue, we assign it back to stage and task
  for (const stage of result.stageList) {
    stage.pipeline = result;
    for (const task of stage.taskList) {
      task.pipeline = result;
      task.stage = stage;
    }
  }

  return result;
}

const getters = {
  convert:
    (state: PipelineState, getters: any, rootState: any, rootGetters: any) =>
    (pipeline: ResourceObject, includedList: ResourceObject[]): Pipeline => {
      return convert(pipeline, includedList, rootGetters);
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
