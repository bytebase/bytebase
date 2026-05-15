<template>
  <ReactPageMount
    page="DashboardBodyShell"
    :page-props="{
      variant: 'workspace',
      isRootPath,
      routeKey: route.fullPath,
      onReady: handleReady,
    }"
  />

  <teleport v-if="sidebarTarget && !isRootPath" :to="sidebarTarget">
    <router-view name="leftSidebar" />
  </teleport>

  <teleport v-if="contentTarget && isProjectRoute" :to="contentTarget">
    <router-view name="content" />
  </teleport>

  <teleport v-else-if="contentTarget" :to="contentTarget">
    <ReactPageMount
      page="RoutePermissionGuardShell"
      :page-props="{
        routeKey: route.fullPath,
        className: 'm-4',
        targetClassName: 'h-full min-h-0',
        onReady: handlePermissionReady,
      }"
    />
  </teleport>

  <teleport
    v-if="!isProjectRoute && routePermissionTarget"
    :to="routePermissionTarget"
  >
    <router-view name="content" />
  </teleport>

  <teleport v-if="quickstartTarget" :to="quickstartTarget">
    <ReactPageMount page="Quickstart" container-class="w-full" />
  </teleport>

</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@/plugins/i18n";
import type { DashboardShellTargets } from "@/react/dashboard-shell";
import ReactPageMount from "@/react/ReactPageMount.vue";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useActuatorV1Store } from "@/store";

import { provideBodyLayoutContext } from "./common";

const actuatorStore = useActuatorV1Store();
const route = useRoute();
const router = useRouter();
const { width: windowWidth } = useWindowSize();

const shellTargets = shallowRef<DashboardShellTargets>({
  desktopSidebar: null,
  mobileSidebar: null,
  content: null,
  quickstart: null,
  mainContainer: null,
});
const mainContainerRef = ref<HTMLDivElement>();
const routePermissionTarget = shallowRef<HTMLDivElement | null>(null);

const isRootPath = computed(() => {
  return router.currentRoute.value.name === WORKSPACE_ROOT_MODULE;
});
const isProjectRoute = computed(() => {
  return Boolean(route.params.projectId);
});

const contentTarget = computed(() => shellTargets.value.content);
const quickstartTarget = computed(() => shellTargets.value.quickstart);
const sidebarTarget = computed(() => {
  if (windowWidth.value >= 768) {
    return shellTargets.value.desktopSidebar;
  }
  return shellTargets.value.mobileSidebar;
});

const handleReady = (targets: DashboardShellTargets) => {
  shellTargets.value = targets;
  mainContainerRef.value = targets.mainContainer ?? undefined;
};

const handlePermissionReady = (target: HTMLDivElement | null) => {
  routePermissionTarget.value = target;
};

watch(
  () => route.fullPath,
  () => {
    routePermissionTarget.value = null;
  },
  { flush: "sync" }
);

const refreshRemindTimer = ref<ReturnType<typeof setTimeout>>();
onMounted(async () => {
  const remind = await actuatorStore.tryToRemindRefresh();
  if (remind) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("refresh-remind.title"),
      description: t("refresh-remind.description"),
      manualHide: true,
    });
  }
  refreshRemindTimer.value = setInterval(
    async () => {
      const remind = await actuatorStore.tryToRemindRefresh();
      if (remind) {
        pushNotification({
          module: "bytebase",
          style: "WARN",
          title: t("refresh-remind.title"),
          description: t("refresh-remind.description"),
          manualHide: true,
        });
      }
    },
    1000 * 60 * 30
  );
});
onUnmounted(() => {
  if (refreshRemindTimer.value) {
    clearInterval(refreshRemindTimer.value);
  }
});

const agentShortcutHandler = async (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "A") {
    e.preventDefault();
    const { useAgentStore } = await import("@/react/plugins/agent/store/agent");
    useAgentStore.getState().toggle();
  }
};
onMounted(() => window.addEventListener("keydown", agentShortcutHandler));
onUnmounted(() => window.removeEventListener("keydown", agentShortcutHandler));

provideBodyLayoutContext({
  mainContainerRef,
});
</script>
