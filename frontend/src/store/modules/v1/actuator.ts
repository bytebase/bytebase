import type { RemovableRef } from "@vueuse/core";
import { useLocalStorage } from "@vueuse/core";
import axios from "axios";
import { defineStore } from "pinia";
import semver from "semver";
import { computed } from "vue";
import { actuatorServiceClient } from "@/grpcweb";
import { useSilentRequest } from "@/plugins/silent-request";
import {
  defaultAppProfile,
  type AppFeatures,
  type AppProfile,
  type Release,
  type ReleaseInfo,
} from "@/types";
import type {
  ActuatorInfo,
  ResourcePackage,
} from "@/types/proto/v1/actuator_service";
import { PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";
import { semverCompare } from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://www.bytebase.com/docs/get-started/install/external-url";
const GITHUB_API_LIST_BYTEBASE_RELEASE =
  "https://api.github.com/repos/bytebase/bytebase/releases";

interface ActuatorState {
  // Whether the app is initialized or not.
  initialized: boolean;
  serverInfo?: ActuatorInfo;
  resourcePackage?: ResourcePackage;
  releaseInfo: RemovableRef<ReleaseInfo>;
  appProfile: AppProfile;
}

export const useActuatorV1Store = defineStore("actuator_v1", {
  state: (): ActuatorState => ({
    initialized: false,
    serverInfo: undefined,
    resourcePackage: undefined,
    releaseInfo: useLocalStorage("bytebase_release", {
      ignoreRemindModalTillNextRelease: false,
      nextCheckTs: 0,
    }),
    appProfile: defaultAppProfile(),
  }),
  getters: {
    changelogURL: (state) => {
      // valid version should following semantic version like 3.1.0
      const version = semver.valid(state.serverInfo?.version);
      if (!version) {
        return "";
      }
      return `https://bytebase.com/changelog/bytebase-${version.split(".").join("-")}/`;
    },
    info: (state) => {
      return state.serverInfo;
    },
    brandingLogo: (state) => {
      if (!state.resourcePackage?.logo) {
        return "";
      }
      return new TextDecoder().decode(state.resourcePackage?.logo);
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
    isDocker: (state) => {
      return state.serverInfo?.docker || false;
    },
    isSaaSMode: (state) => {
      return state.serverInfo?.saas || false;
    },
    needAdminSetup: (state) => {
      return state.serverInfo?.needAdminSetup || false;
    },
    needConfigureExternalUrl: (state) => {
      if (!state.serverInfo) return false;
      const url = state.serverInfo?.externalUrl ?? "";
      return url === "" || url === EXTERNAL_URL_PLACEHOLDER;
    },
    disallowSignup: (state) => {
      return state.serverInfo?.disallowSignup || false;
    },
    disallowPasswordSignin: (state) => {
      return state.serverInfo?.disallowPasswordSignin || false;
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
    passwordRestriction: (state) => {
      return (
        state.serverInfo?.passwordRestriction ??
        PasswordRestrictionSetting.fromPartial({
          minLength: 8,
          requireLetter: true,
        })
      );
    },
  },
  actions: {
    setLogo(logo: string) {
      if (this.resourcePackage) {
        this.resourcePackage.logo = new TextEncoder().encode(logo);
      }
    },
    setServerInfo(serverInfo: ActuatorInfo) {
      this.serverInfo = serverInfo;
    },
    async fetchServerInfo() {
      const [serverInfo, resourcePackage] = await Promise.all([
        actuatorServiceClient.getActuatorInfo({}),
        actuatorServiceClient.getResourcePackage({}),
      ]);
      this.setServerInfo(serverInfo);
      this.resourcePackage = resourcePackage;
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
    async tryToRemindRelease(): Promise<boolean> {
      if (this.serverInfo?.saas ?? false) {
        return false;
      }
      if (!this.releaseInfo.latest) {
        const release = await this.fetchLatestRelease();
        this.releaseInfo.latest = release;
      }
      if (!this.releaseInfo.latest) {
        return false;
      }

      // It's time to fetch the release
      if (new Date().getTime() >= this.releaseInfo.nextCheckTs) {
        const release = await this.fetchLatestRelease();
        if (!release) {
          return false;
        }

        // check till 24 hours later
        this.releaseInfo.nextCheckTs =
          new Date().getTime() + 24 * 60 * 60 * 1000;

        if (semverCompare(release.tag_name, this.releaseInfo.latest.tag_name)) {
          this.releaseInfo.ignoreRemindModalTillNextRelease = false;
        }

        this.releaseInfo.latest = release;
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
    overrideAppFeatures(overrides: Partial<AppFeatures>) {
      Object.assign(this.appProfile.features, overrides);
    },
  },
});

export const useAppFeature = <T extends keyof AppFeatures>(feature: T) => {
  return computed(() => useActuatorV1Store().appProfile.features[feature]);
};
