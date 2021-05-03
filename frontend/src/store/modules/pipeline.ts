import axios from "axios";
import {
  ResourceIdentifier,
  ResourceObject,
  Pipeline,
  PipelineId,
  PipelineState,
  PipelineStatusPatch,
  Task,
  Issue,
  IssueId,
  unknown,
  Stage,
} from "../../types";

const state: () => PipelineState = () => ({});

function convert(
  pipeline: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Pipeline {
  const creator = rootGetters["principal/principalById"](
    pipeline.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    pipeline.attributes.updaterId
  );

  const stageList: Stage[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "stage" &&
      (item.relationships!.pipeline.data as ResourceIdentifier).id ==
        pipeline.id
    ) {
      const stage: Stage = rootGetters["stage/convertPartial"](
        item,
        includedList
      );
      stageList.push(stage);
    }
  }

  const result: Pipeline = {
    ...(pipeline.attributes as Omit<
      Pipeline,
      "id" | "creator" | "updater" | "stageList"
    >),
    id: pipeline.id,
    creator,
    updater,
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
  convert: (
    state: PipelineState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (pipeline: ResourceObject, includedList: ResourceObject[]): Pipeline => {
    return convert(pipeline, includedList, rootGetters);
  },

  async updatePipelineStatus(
    { dispatch }: any,
    {
      pipelineId,
      pipelineStatusPatch,
    }: {
      pipelineId: PipelineId;
      pipelineStatusPatch: PipelineStatusPatch;
    }
  ) {
    const data = (
      await axios.patch(`/mock/pipeline/${pipelineId}/status`, {
        data: {
          type: "pipelinestatuspatch",
          attributes: pipelineStatusPatch,
        },
      })
    ).data;
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
