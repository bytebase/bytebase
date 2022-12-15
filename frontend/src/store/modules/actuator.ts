import axios from "axios";
import { defineStore } from "pinia";
import { ActuatorState, ServerInfo, Release } from "@/types";
import { useLocalStorage } from "@vueuse/core";
import { semverCompare } from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://www.bytebase.com/docs/get-started/install/external-url";
const GITHUB_API_LIST_BYTEBASE_RELEASE =
  "https://api.github.com/repos/bytebase/bytebase/releases";

export const useActuatorStore = defineStore("actuator", {
  state: (): ActuatorState => ({
    serverInfo: undefined,
    releaseInfo: useLocalStorage("bytebase_release", {
      ignoreRemindModalTillNextRelease: false,
      nextCheckTs: 0,
    }),
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
    hasNewRelease: (state) => {
      return (
        (state.serverInfo?.version === "development" &&
          !!state.releaseInfo.lastest?.tag_name) ||
        semverCompare(
          state.releaseInfo.lastest?.tag_name ?? "",
          state.serverInfo?.version ?? ""
        )
      );
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
    async tryToRemindRelease(): Promise<boolean> {
      if (!this.releaseInfo.lastest) {
        const relase = await this.fetchLastestRelease();
        this.releaseInfo.lastest = relase;
      }
      if (!this.releaseInfo.lastest) {
        return false;
      }

      // It's time to fetch the release
      if (new Date().getTime() >= this.releaseInfo.nextCheckTs) {
        const relase = await this.fetchLastestRelease();
        if (!relase) {
          return false;
        }

        // check till 24 hours later
        this.releaseInfo.nextCheckTs =
          new Date().getTime() + 24 * 60 * 60 * 1000;

        if (semverCompare(relase.tag_name, this.releaseInfo.lastest.tag_name)) {
          this.releaseInfo.ignoreRemindModalTillNextRelease = false;
        }

        this.releaseInfo.lastest = relase;
      }

      if (this.releaseInfo.ignoreRemindModalTillNextRelease) {
        return false;
      }

      return this.hasNewRelease;
    },
    async fetchLastestRelease(): Promise<Release | undefined> {
      try {
        const { data: releaseList } = await axios.get<Release[]>(
          `${GITHUB_API_LIST_BYTEBASE_RELEASE}?per_page=1`
        );
        return releaseList[0];
      } catch {
        // It's okay to ignore the failure and just return undefined.
        return;
      }
    },
  },
});
