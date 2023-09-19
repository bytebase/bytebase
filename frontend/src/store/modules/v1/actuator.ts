import { RemovableRef } from "@vueuse/core";
import { useLocalStorage } from "@vueuse/core";
import axios from "axios";
import { defineStore, storeToRefs } from "pinia";
import { watchEffect } from "vue";
import { actuatorServiceClient } from "@/grpcweb";
import { useSilentRequest } from "@/plugins/silent-request";
import { Release, ReleaseInfo } from "@/types";
import { ActuatorInfo, DebugLog } from "@/types/proto/v1/actuator_service";
import { semverCompare } from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://www.bytebase.com/docs/get-started/install/external-url";
const GITHUB_API_LIST_BYTEBASE_RELEASE =
  "https://api.github.com/repos/bytebase/bytebase/releases";

interface ActuatorState {
  serverInfo?: ActuatorInfo;
  releaseInfo: RemovableRef<ReleaseInfo>;
  debugLogList: DebugLog[];
}

export const useActuatorV1Store = defineStore("actuator_v1", {
  state: (): ActuatorState => ({
    serverInfo: undefined,
    releaseInfo: useLocalStorage("bytebase_release", {
      ignoreRemindModalTillNextRelease: false,
      nextCheckTs: 0,
    }),
    debugLogList: [],
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
      return state.serverInfo?.demoName;
    },
    isReadonly: (state) => {
      return state.serverInfo?.readonly || false;
    },
    isDebug: (state) => {
      return state.serverInfo?.debug || false;
    },
    isSaaSMode: (state) => {
      return state.serverInfo?.saas || false;
    },
    needAdminSetup: (state) => {
      return state.serverInfo?.needAdminSetup || false;
    },
    needConfigureExternalUrl: (state) => {
      const url = state.serverInfo?.externalUrl ?? "";
      return url === "" || url === EXTERNAL_URL_PLACEHOLDER;
    },
    disallowSignup: (state) => {
      return state.serverInfo?.disallowSignup || false;
    },
    hasNewRelease: (state) => {
      return (
        (state.serverInfo?.version === "development" &&
          !!state.releaseInfo.latest?.tag_name) ||
        semverCompare(
          state.releaseInfo.latest?.tag_name ?? "",
          state.serverInfo?.version ?? ""
        )
      );
    },
  },
  actions: {
    setServerInfo(serverInfo: ActuatorInfo) {
      this.serverInfo = serverInfo;
    },
    async fetchServerInfo() {
      const serverInfo = await actuatorServiceClient.getActuatorInfo({});
      this.setServerInfo(serverInfo);

      return serverInfo;
    },
    async patchDebug({ debug }: { debug: boolean }) {
      const serverInfo = await actuatorServiceClient.updateActuatorInfo({
        actuator: {
          debug,
        },
        updateMask: ["debug"],
      });
      this.setServerInfo(serverInfo);
    },
    async fetchDebugLogList() {
      const { logs } = await actuatorServiceClient.listDebugLog({});
      this.debugLogList = logs;
      return logs;
    },
    async tryToRemindRelease(): Promise<boolean> {
      if (this.serverInfo?.saas ?? false) {
        return false;
      }
      if (!this.releaseInfo.latest) {
        const relase = await this.fetchLatestRelease();
        this.releaseInfo.latest = relase;
      }
      if (!this.releaseInfo.latest) {
        return false;
      }

      // It's time to fetch the release
      if (new Date().getTime() >= this.releaseInfo.nextCheckTs) {
        const relase = await this.fetchLatestRelease();
        if (!relase) {
          return false;
        }

        // check till 24 hours later
        this.releaseInfo.nextCheckTs =
          new Date().getTime() + 24 * 60 * 60 * 1000;

        if (semverCompare(relase.tag_name, this.releaseInfo.latest.tag_name)) {
          this.releaseInfo.ignoreRemindModalTillNextRelease = false;
        }

        this.releaseInfo.latest = relase;
      }

      if (this.releaseInfo.ignoreRemindModalTillNextRelease) {
        return false;
      }

      return this.hasNewRelease;
    },
    async fetchLatestRelease(): Promise<Release | undefined> {
      try {
        const { data: releaseList } = await useSilentRequest(() =>
          axios.get<Release[]>(`${GITHUB_API_LIST_BYTEBASE_RELEASE}?per_page=1`)
        );
        return releaseList[0];
      } catch {
        // It's okay to ignore the failure and just return undefined.
        return;
      }
    },
  },
});

export const useDebugLogList = () => {
  const store = useActuatorV1Store();
  watchEffect(() => store.fetchDebugLogList());

  return storeToRefs(store).debugLogList;
};
