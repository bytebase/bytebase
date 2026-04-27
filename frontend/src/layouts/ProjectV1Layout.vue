<template>
  <template v-if="initialized">
    <ArchiveBanner v-if="project.state === State.DELETED" class="py-2 mb-4" />
    <div v-if="isDefaultProject" class="m-4">
      <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
        {{ $t("database.unassigned-databases") }}
      </h1>
      <BBAttention class="mb-4" type="info">
        {{ $t("project.overview.info-slot-content") }}
      </BBAttention>
    </div>

    <ReactPageMount
      page="RoutePermissionGuardShell"
      :page-props="{
        project,
        className: 'm-4',
        targetClassName: 'h-full min-h-0',
        onReady: handlePermissionReady,
      }"
    />
    <teleport
      v-if="routePermitted && routePermissionTarget"
      :to="routePermissionTarget"
    >
      <router-view
        :project-id="projectId"
        :allow-edit="allowEdit"
        v-bind="$attrs"
      />
    </teleport>
  </template>
  <div
    v-else
    class="fixed inset-0 bg-white flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>

  <IAMRemindModal :project-name="project.name" />
</template>

<script lang="tsx" setup>
import { type ConnectError } from "@connectrpc/connect";
import { NSpin } from "naive-ui";
import { computed, shallowRef, watch, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import IAMRemindModal from "@/components/IAMRemindModal.vue";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import { useRoutePermitted } from "@/composables/useRoutePermitted";
import ReactPageMount from "@/react/ReactPageMount.vue";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  pushNotification,
  useProjectByName,
  useProjectIamPolicy,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  isDefaultProject as checkIsDefaultProject,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { hasProjectPermissionV2, setDocumentTitle } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const router = useRouter();
const recentProjects = useRecentProjects();
const projectStore = useProjectV1Store();
const { remove: removeVisit } = useRecentVisit();
const routePermissionTarget = shallowRef<HTMLDivElement | null>(null);

const handlePermissionReady = (target: HTMLDivElement | null) => {
  routePermissionTarget.value = target;
};

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);

watchEffect(async () => {
  // Capture the current project name before the async gap so we can
  // detect stale responses if projectId changes while the fetch is
  // in flight.
  const currentName = projectName.value;
  try {
    const project = await projectStore.getOrFetchProjectByName(
      currentName,
      false
    );
    // If projectId changed during the fetch, ignore this stale result.
    if (currentName !== projectName.value) return;
    // If the project resolves to the unknown sentinel (e.g. projectId is "-1"),
    // it will never initialize. Redirect to the landing page instead of
    // spinning forever.
    if (project.name === UNKNOWN_PROJECT_NAME) {
      const projectRoute = router.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId: props.projectId },
      });
      removeVisit(projectRoute.fullPath);
      router.replace({ name: WORKSPACE_ROUTE_LANDING });
      return;
    }
    recentProjects.setRecentProject(project.name);
  } catch (err) {
    if (currentName !== projectName.value) return;
    console.error(err);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to fetch project ${props.projectId}`,
      description: (err as ConnectError).message,
    });

    const projectRoute = router.resolve({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: {
        projectId: props.projectId,
      },
    });
    removeVisit(projectRoute.fullPath);
    router.replace({
      name: WORKSPACE_ROUTE_LANDING,
    });
  }
});

const { project, ready } = useProjectByName(projectName);
const { ready: iamReady } = useProjectIamPolicy(projectName);
const routePermitted = useRoutePermitted(project);

const initialized = computed(
  () =>
    ready.value && iamReady.value && project.value.name !== UNKNOWN_PROJECT_NAME
);

const isDefaultProject = computed((): boolean => {
  return checkIsDefaultProject(project.value.name);
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(project.value, "bb.projects.update");
});

const route = useRoute();
watch(
  [() => route.meta.title, () => project.value.title, initialized],
  () => {
    if (!initialized.value) return;
    const pageTitle = route.meta.title ? route.meta.title(route) : undefined;
    if (pageTitle) {
      setDocumentTitle(pageTitle, project.value.title);
    } else {
      setDocumentTitle(project.value.title);
    }
  },
  { immediate: true }
);
</script>
