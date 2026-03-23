import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import semver from "semver";
import { computed, ref } from "vue";
import { actuatorServiceClientConnect } from "@/connect";
import {
  type AppFeatures,
  type AppProfile,
  defaultAppProfile,
  type Release,
  type ReleaseInfo,
} from "@/types";
import type {
  ActuatorInfo,
  ResourcePackage,
} from "@/types/proto-es/v1/actuator_service_pb";
import {
  STORAGE_KEY_ONBOARDING,
  STORAGE_KEY_RELEASE,
  semverCompare,
} from "@/utils";

const EXTERNAL_URL_PLACEHOLDER =
  "https://docs.bytebase.com/get-started/self-host/external-url";
const GITHUB_API_LIST_BYTEBASE_RELEASE =
  "https://api.github.com/repos/bytebase/bytebase/releases";

export const useActuatorV1Store = defineStore("actuator_v1", () => {
  // State
  const initialized = ref(false);
  const serverInfo = ref<ActuatorInfo | undefined>(undefined);
  const serverInfoTs = ref(0);
  const resourcePackage = ref<ResourcePackage | undefined>(undefined);
  const releaseInfo = useLocalStorage<ReleaseInfo>(STORAGE_KEY_RELEASE, {
    ignoreRemindModalTillNextRelease: false,
    nextCheckTs: 0,
  });
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

  const brandingLogo = computed(() => {
    if (!resourcePackage.value?.logo) {
      return "";
    }
    return new TextDecoder().decode(resourcePackage.value?.logo);
  });

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

  const hasNewRelease = computed(() => {
    return (
      (serverInfo.value?.version === "development" &&
        !!releaseInfo.value.latest?.tag_name) ||
      semverCompare(
        releaseInfo.value.latest?.tag_name ?? "",
        serverInfo.value?.version ?? ""
      )
    );
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

  const setLogo = (logo: string) => {
    if (resourcePackage.value) {
      resourcePackage.value.logo = new TextEncoder().encode(logo);
    }
  };

  const setServerInfo = (info: ActuatorInfo) => {
    serverInfo.value = info;
    serverInfoTs.value = Date.now();
  };

  const fetchActuatorInfo = async (workspace?: string) => {
    if (!workspace) {
      return actuatorServiceClientConnect.getActuatorInfo({});
    }
    return actuatorServiceClientConnect.getWorkspaceActuatorInfo({
      name: workspace,
    });
  };

  const fetchServerInfo = async (workspace?: string) => {
    const [info, pkg] = await Promise.all([
      fetchActuatorInfo(workspace),
      actuatorServiceClientConnect.getResourcePackage({
        name: workspace,
      }),
    ]);
    setServerInfo(info);
    resourcePackage.value = pkg;
    return info;
  };

  const fetchLatestRelease = async (): Promise<Release | undefined> => {
    try {
      const response = await fetch(
        `${GITHUB_API_LIST_BYTEBASE_RELEASE}?per_page=1`
      );
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const releaseList = (await response.json()) as Release[];
      return releaseList[0];
    } catch {
      return;
    }
  };

  const tryToRemindRelease = async (): Promise<boolean> => {
    if (serverInfo.value?.saas ?? false) {
      return false;
    }
    if (!releaseInfo.value.latest) {
      const release = await fetchLatestRelease();
      releaseInfo.value.latest = release;
    }
    if (!releaseInfo.value.latest) {
      return false;
    }

    if (Date.now() >= releaseInfo.value.nextCheckTs) {
      const release = await fetchLatestRelease();
      if (!release) {
        return false;
      }

      releaseInfo.value.nextCheckTs = Date.now() + 24 * 60 * 60 * 1000;

      if (semverCompare(release.tag_name, releaseInfo.value.latest.tag_name)) {
        releaseInfo.value.ignoreRemindModalTillNextRelease = false;
      }

      releaseInfo.value.latest = release;
    }

    if (releaseInfo.value.ignoreRemindModalTillNextRelease) {
      return false;
    }

    return hasNewRelease.value;
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
    resourcePackage,
    releaseInfo,
    appProfile,
    onboardingState,
    // Getters
    changelogURL,
    info,
    brandingLogo,
    version,
    gitCommitBE,
    gitCommitFE,
    isDemo,
    isDocker,
    isSaaSMode,
    workspaceResourceName,
    needConfigureExternalUrl,
    hasNewRelease,
    activatedInstanceCount,
    totalInstanceCount,
    replicaCount,
    quickStartEnabled,
    enableOnboarding,
    activeUserCount,
    // Actions
    updateUserStat,
    setLogo,
    setServerInfo,
    fetchServerInfo,
    fetchLatestRelease,
    tryToRemindRelease,
    tryToRemindRefresh,
    setupSample,
    overrideAppFeatures,
  };
});

export const useAppFeature = <T extends keyof AppFeatures>(feature: T) => {
  return computed(() => useActuatorV1Store().appProfile.features[feature]);
};
