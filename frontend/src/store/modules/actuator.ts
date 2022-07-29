import axios from "axios";
import { defineStore } from "pinia";
import { ActuatorState, ServerInfo } from "@/types";

export const useActuatorStore = defineStore("actuator", {
  state: (): ActuatorState => ({
    serverInfo: undefined,
  }),
  getters: {
    info: (state) => {
      return state.serverInfo;
    },
    version: (state) => {
      return state.serverInfo?.version || "";
    },
    gitCommit: (state) => {
      return state.serverInfo?.gitCommit || "";
    },
    isDemo: (state) => {
      return state.serverInfo?.demo || false;
    },
    isReadonly: (state) => {
      return state.serverInfo?.readonly || false;
    },
    needAdminSetup: (state) => {
      return state.serverInfo?.needAdminSetup || false;
    },
  },
  actions: {
    setServerInfo(serverInfo: ServerInfo) {
      this.serverInfo = serverInfo;
    },
    async fetchServerInfo() {
      const serverInfo = (await axios.get(`/api/actuator/info`))
        .data as ServerInfo;

      this.setServerInfo(serverInfo);

      return serverInfo;
    },
  },
});
