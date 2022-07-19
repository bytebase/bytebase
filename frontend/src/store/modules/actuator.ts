import { defineStore } from "pinia";
import axios from "axios";
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
    isLiveDemo: (state) => {
      return (
        state.serverInfo?.demo &&
        state.serverInfo?.demoName !== "" &&
        state.serverInfo?.demoName !== "dev" &&
        state.serverInfo?.demoName !== "prod"
      );
    },
  },
  actions: {
    async fetchInfo() {
      const serverInfo = (await axios.get(`/api/actuator/info`))
        .data as ServerInfo;

      this.setInfo(serverInfo);

      return serverInfo;
    },
    setInfo(serverInfo: ServerInfo) {
      this.serverInfo = serverInfo;
    },
  },
});
