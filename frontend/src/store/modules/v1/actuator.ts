import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import semver from "semver";
import { computed, ref } from "vue";
import { actuatorServiceClientConnect } from "@/connect";
import { type AppFeatures, type AppProfile, defaultAppProfile } from "@/types";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import { STORAGE_KEY_ONBOARDING } from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://docs.bytebase.com/get-started/self-host/external-url";

export const useActuatorV1Store = defineStore("actuator_v1", () => {
  // State
  const initialized = ref(false);
  const serverInfo = ref<ActuatorInfo | undefined>(undefined);
  const serverInfoTs = ref(0);
  const appProfile = ref<AppProfile>(defaultAppProfile());
  const onboardingState = useLocalStorage<{
    consumed: string[];
  }>(STORAGE_KEY_ONBOARDING, {
    consumed: [],
  });

  // Getters
  const changelogURL = computed(() => {
    const version = semver.valid(serverInfo.value?.version);
    if (!version) {
      return "";
    }
    return `https://docs.bytebase.com/changelog/bytebase-${version.split(".").join("-")}/`;
  });

  const info = computed(() => serverInfo.value);

  const version = computed(() => serverInfo.value?.version || "");

  const gitCommitBE = computed(() => {
    const commit = serverInfo.value?.gitCommit ?? "";
    return commit === "" ? "unknown" : commit;
  });

  const gitCommitFE = computed(() => {
    const commit = import.meta.env.GIT_COMMIT ?? "";
    return commit === "" ? "unknown" : commit;
  });

  const isDemo = computed(() => serverInfo.value?.demo);

  const isDocker = computed(() => serverInfo.value?.docker || false);

  const isSaaSMode = computed(() => serverInfo.value?.saas || false);

  const workspaceResourceName = computed(
    () => serverInfo.value?.workspace ?? ""
  );

  const needConfigureExternalUrl = computed(() => {
    if (!serverInfo.value) return false;
    const url = serverInfo.value?.externalUrl ?? "";
    return url === "" || url === EXTERNAL_URL_PLACEHOLDER;
  });

  const activatedInstanceCount = computed(
    () => serverInfo.value?.activatedInstanceCount ?? 0
  );

  const totalInstanceCount = computed(
    () => serverInfo.value?.totalInstanceCount ?? 0
  );

  const replicaCount = computed(() => serverInfo.value?.replicaCount ?? 1);

  const activeUserCount = computed(
    () => serverInfo.value?.activatedUserCount ?? 0
  );

  const userCountInIam = computed(() => serverInfo.value?.userCountInIam ?? 0);

  const enableOnboarding = computed(() => {
    return activeUserCount.value === 1 && !isSaaSMode.value;
  });

  const updateUserStat = (count: number) => {
    if (!serverInfo.value) {
      return;
    }
    serverInfo.value.activatedUserCount += count;
    serverInfo.value.activatedUserCount = Math.max(
      0,
      serverInfo.value.activatedUserCount
    );
  };

  const quickStartEnabled = computed(() => {
    if (useAppFeature("bb.feature.hide-quick-start").value) {
      return false;
    }
    if (!serverInfo.value?.enableSample) {
      return false;
    }

    // Hide quickstart if there are more than 1 active users.
    return activeUserCount.value <= 1;
  });

  const setServerInfo = (info: ActuatorInfo) => {
    serverInfo.value = info;
    serverInfoTs.value = Date.now();
  };

  const fetchActuatorInfo = async (workspace?: string) => {
    return actuatorServiceClientConnect.getActuatorInfo({
      name: workspace ?? "",
    });
  };

  const fetchServerInfo = async (workspace?: string) => {
    const info = await fetchActuatorInfo(workspace);
    setServerInfo(info);
    return info;
  };

  const tryToRemindRefresh = async (): Promise<boolean> => {
    if (Date.now() - serverInfoTs.value >= 1000 * 60 * 30) {
      await fetchServerInfo(workspaceResourceName.value);
    }
    if (gitCommitBE.value === "unknown" || gitCommitFE.value === "unknown") {
      return false;
    }
    return gitCommitBE.value !== gitCommitFE.value;
  };

  const setupSample = async () => {
    await actuatorServiceClientConnect.setupSample({});
  };

  const overrideAppFeatures = (overrides: Partial<AppFeatures>) => {
    Object.assign(appProfile.value.features, overrides);
  };

  return {
    // State
    initialized,
    serverInfo,
    serverInfoTs,
    appProfile,
    onboardingState,
    // Getters
    changelogURL,
    info,
    version,
    gitCommitBE,
    gitCommitFE,
    isDemo,
    isDocker,
    isSaaSMode,
    workspaceResourceName,
    needConfigureExternalUrl,
    activatedInstanceCount,
    totalInstanceCount,
    replicaCount,
    quickStartEnabled,
    enableOnboarding,
    activeUserCount,
    userCountInIam,
    // Actions
    updateUserStat,
    setServerInfo,
    fetchServerInfo,
    tryToRemindRefresh,
    setupSample,
    overrideAppFeatures,
  };
});

export const useAppFeature = <T extends keyof AppFeatures>(feature: T) => {
  return computed(() => useActuatorV1Store().appProfile.features[feature]);
};
