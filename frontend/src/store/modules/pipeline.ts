import axios from "axios";
import {
  ResourceIdentifier,
  ResourceObject,
  Pipeline,
  PipelineId,
  PipelineState,
  PipelineStatusPatch,
  Step,
  Issue,
  IssueId,
  unknown,
  Task,
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

  const taskList: Task[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "task" &&
      (item.relationships!.pipeline.data as ResourceIdentifier).id ==
        pipeline.id
    ) {
      const task: Task = rootGetters["task/convertPartial"](item, includedList);
      taskList.push(task);
    }
  }

  const result: Pipeline = {
    ...(pipeline.attributes as Omit<
      Pipeline,
      "id" | "creator" | "updater" | "taskList"
    >),
    id: pipeline.id,
    creator,
    updater,
    taskList,
  };

  // Now we have a complate issue, we assign it back to task and step
  for (const task of result.taskList) {
    task.pipeline = result;
    for (const step of task.stepList) {
      step.pipeline = result;
      step.task = task;
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
      await axios.patch(`/api/pipeline/${pipelineId}/status`, {
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
