import axios from "axios";
import { defineStore } from "pinia";
import { ActuatorState, ServerInfo } from "@/types";

const EXTERNAL_URL_PLACEHOLDER =
  "https://www.bytebase.com/docs/get-started/install/external-url";

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
    needConfigureExternalUrl: (state) => {
      return state.serverInfo?.externalUrl === EXTERNAL_URL_PLACEHOLDER;
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
