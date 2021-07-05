import axios from "axios";
import { ActuatorState, ServerInfo } from "../../types";

const state: () => ActuatorState = () => ({
  serverInfo: undefined,
});

const getters = {
  info: (state: ActuatorState) => (): ServerInfo | undefined => {
    return state.serverInfo;
  },
  version: (state: ActuatorState) => (): string => {
    return state.serverInfo?.version || "";
  },
  isDemo: (state: ActuatorState) => (): boolean => {
    return state.serverInfo?.demo || false;
  },
  isReadonly: (state: ActuatorState) => (): boolean => {
    return state.serverInfo?.readonly || false;
  },
  needAdminSetup: (state: ActuatorState) => (): boolean => {
    return state.serverInfo?.needAdminSetup || false;
  },
};

const actions = {
  async fetchInfo({ commit }: any): Promise<ServerInfo> {
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
