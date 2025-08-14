import { create } from "@bufbuild/protobuf";
import type { RemovableRef } from "@vueuse/core";
import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import semver from "semver";
import { computed } from "vue";
import { actuatorServiceClientConnect } from "@/grpcweb";
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
} from "@/types/proto-es/v1/actuator_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { PasswordRestrictionSettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import { semverCompare } from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://docs.bytebase.com/get-started/self-host/external-url";
const GITHUB_API_LIST_BYTEBASE_RELEASE =
  "https://api.github.com/repos/bytebase/bytebase/releases";

interface ActuatorState {
  // Whether the app is initialized or not.
  initialized: boolean;
  serverInfo?: ActuatorInfo;
  // Last time when we fetched the server info.
  serverInfoTs: number;
  resourcePackage?: ResourcePackage;
  releaseInfo: RemovableRef<ReleaseInfo>;
  appProfile: AppProfile;
  onboardingState: RemovableRef<{
    isOnboarding: boolean;
    consumed: string[];
  }>;
}

export const useActuatorV1Store = defineStore("actuator_v1", {
  state: (): ActuatorState => ({
    initialized: false,
    serverInfo: undefined,
    serverInfoTs: 0,
    resourcePackage: undefined,
    releaseInfo: useLocalStorage("bytebase_release", {
      ignoreRemindModalTillNextRelease: false,
      nextCheckTs: 0,
    }),
    appProfile: defaultAppProfile(),
    onboardingState: useLocalStorage<{
      isOnboarding: boolean;
      consumed: string[];
    }>("bb.onboarding-state", {
      isOnboarding: false,
      consumed: [],
    }),
  }),
  getters: {
    changelogURL: (state) => {
      // valid version should following semantic version like 3.1.0
      const version = semver.valid(state.serverInfo?.version);
      if (!version) {
        return "";
      }
      return `https://docs.bytebase.com/changelog/bytebase-${version.split(".").join("-")}/`;
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
    gitCommitBE: (state) => {
      const commit = state.serverInfo?.gitCommit ?? "";
      return commit === "" ? "unknown" : commit;
    },
    gitCommitFE: () => {
      const commit = import.meta.env.GIT_COMMIT ?? "";
      return commit === "" ? "unknown" : commit;
    },
    isDemo: (state) => {
      return state.serverInfo?.demo;
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
        create(PasswordRestrictionSettingSchema, {
          minLength: 8,
          requireLetter: true,
        })
      );
    },
    activatedInstanceCount: (state) => {
      return state.serverInfo?.activatedInstanceCount ?? 0;
    },
    totalInstanceCount: (state) => {
      return state.serverInfo?.totalInstanceCount ?? 0;
    },
    inactiveUserCount: (state) => {
      return (state.serverInfo?.userStats ?? []).reduce((count, stat) => {
        if (stat.state === State.DELETED) {
          count += stat.count;
        }
        return count;
      }, 0);
    },
  },
  actions: {
    getActiveUserCount({ includeBot }: { includeBot: boolean }) {
      return (this.serverInfo?.userStats ?? []).reduce((count, stat) => {
        if (stat.state !== State.ACTIVE) {
          return count;
        }
        if (!includeBot && stat.userType === UserType.SYSTEM_BOT) {
          return count;
        }
        count += stat.count;
        return count;
      }, 0);
    },
    setLogo(logo: string) {
      if (this.resourcePackage) {
        this.resourcePackage.logo = new TextEncoder().encode(logo);
      }
    },
    setServerInfo(serverInfo: ActuatorInfo) {
      this.serverInfo = serverInfo;
      this.serverInfoTs = Date.now();
    },
    async fetchServerInfo() {
      const [serverInfo, resourcePackage] = await Promise.all([
        actuatorServiceClientConnect.getActuatorInfo({}),
        actuatorServiceClientConnect.getResourcePackage({}),
      ]);
      this.setServerInfo(serverInfo);
      this.resourcePackage = resourcePackage;
      return serverInfo;
    },
    async patchDebug({ debug }: { debug: boolean }) {
      const serverInfo = await actuatorServiceClientConnect.updateActuatorInfo({
        actuator: {
          debug,
        },
        updateMask: {
          paths: ["debug"],
        },
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
    async tryToRemindRefresh(): Promise<boolean> {
      // refetch after 30 minutes to keep the info fresh.
      if (Date.now() - this.serverInfoTs >= 1000 * 60 * 30) {
        await this.fetchServerInfo();
      }
      if (this.gitCommitBE === "unknown" || this.gitCommitFE === "unknown") {
        return false;
      }
      return this.gitCommitBE !== this.gitCommitFE;
    },
    async fetchLatestRelease(): Promise<Release | undefined> {
      try {
        const releaseList = await useSilentRequest(async () => {
          const response = await fetch(
            `${GITHUB_API_LIST_BYTEBASE_RELEASE}?per_page=1`
          );
          if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
          }
          return response.json() as Promise<Release[]>;
        });
        return releaseList[0];
      } catch {
        // It's okay to ignore the failure and just return undefined.
        return;
      }
    },
    async setupSample() {
      await actuatorServiceClientConnect.setupSample({});
    },
    overrideAppFeatures(overrides: Partial<AppFeatures>) {
      Object.assign(this.appProfile.features, overrides);
    },
  },
});

export const useAppFeature = <T extends keyof AppFeatures>(feature: T) => {
  return computed(() => useActuatorV1Store().appProfile.features[feature]);
};
