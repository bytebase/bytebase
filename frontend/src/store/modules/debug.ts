import axios from "axios";
import { DebugState, ResourceObject } from "../../types";
import { Debug, DebugPatch } from "../../types/debug";

function convert(debug: ResourceObject): Debug {
  return { ...(debug.attributes as Omit<Debug, "">) };
}

const state: () => DebugState = () => ({
  isDebug: false,
});

const getters = {
  isDebug: (state: DebugState) => () => {
    return state.isDebug;
  },
};

const actions = {
  async fetchDebug() {
    const data = (await axios.get("/api/debug")).data;
    return convert(data);
  },
  async patchDebug({ commit }: any, debugPatch: DebugPatch) {
    const debugState: Debug = convert(
      (
        await axios.patch("/api/debug", {
          data: {
            type: "debugPatch",
            attributes: { isDebug: debugPatch.isDebug },
          },
        })
      ).data.data
    );
    commit("setDebug", debugState);
  },
};

const mutations = {
  setDebug(state: DebugState, debug: Debug) {
    state.isDebug = debug.isDebug;
  },
};

export default { namespaced: true, state, getters, actions, mutations };
