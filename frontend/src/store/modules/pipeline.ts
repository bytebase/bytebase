import axios from "axios";
import { UserId, PipelineId, Pipeline, PipelineState } from "../../types";

const state: () => PipelineState = () => ({
  pipelineListByUser: new Map(),
  pipelineById: new Map(),
});

const getters = {
  pipelineListByUser: (state: PipelineState) => (userId: UserId) => {
    return state.pipelineListByUser.get(userId);
  },
  pipelineById: (state: PipelineState) => (pipelineId: PipelineId) => {
    return state.pipelineById.get(pipelineId);
  },
};

const actions = {
  async fetchPipelineListForUser({ commit }: any, userId: UserId) {
    const pipelineList = (await axios.get(`/api/pipeline?userid=${userId}`))
      .data.data;
    commit("setPipelineListForUser", { userId, pipelineList });
    return pipelineList;
  },

  async fetchPipelineById({ commit }: any, pipelineId: PipelineId) {
    const pipeline = (await axios.get(`/api/pipeline/${pipelineId}`)).data.data;
    commit("setPipelineById", {
      pipelineId,
      pipeline,
    });
    return pipeline;
  },
};

const mutations = {
  setPipelineListForUser(
    state: PipelineState,
    {
      userId,
      pipelineList,
    }: {
      userId: UserId;
      pipelineList: Pipeline[];
    }
  ) {
    state.pipelineListByUser.set(userId, pipelineList);
  },
  setPipelineById(
    state: PipelineState,
    {
      pipelineId,
      pipeline,
    }: {
      pipelineId: PipelineId;
      pipeline: Pipeline;
    }
  ) {
    state.pipelineById.set(pipelineId, pipeline);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
