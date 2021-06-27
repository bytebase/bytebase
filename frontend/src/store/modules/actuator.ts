import axios from "axios";
import { ActuatorState, ServerInfo } from "../../types";

const state: () => ActuatorState = () => ({
  serverInfo: undefined,
});

const getters = {
  info: (state: ActuatorState) => (): ServerInfo | undefined => {
    return state.serverInfo;
  },
  isDemo: (state: ActuatorState) => (): boolean => {
    return state.serverInfo?.demo || false;
  },
};

const actions = {
  async info({ commit }: any): Promise<ServerInfo> {
    const serverInfo = (await axios.get(`/api/actuator/info`)).data;

    commit("setInfo", serverInfo);

    return serverInfo;
  },
};

const mutations = {
  setInfo(state: ActuatorState, serverInfo: ServerInfo) {
    state.serverInfo = serverInfo;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
