<template>
  <template v-if="initialized">
    <ArchiveBanner v-if="project.state === State.DELETED" class="py-2 mb-4" />
    <template v-if="isDefaultProject">
      <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
        {{ $t("database.unassigned-databases") }}
      </h1>
      <BBAttention class="mb-4" type="info">
        {{ $t("project.overview.info-slot-content") }}
      </BBAttention>
    </template>

    <RoutePermissionGuard
      :project="project"
      :routes="projectRoutes"
    >
      <router-view
        :project-id="projectId"
        :allow-edit="allowEdit"
        v-bind="$attrs"
      />
    </RoutePermissionGuard>
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
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import IAMRemindModal from "@/components/IAMRemindModal.vue";
import RoutePermissionGuard from "@/components/Permission/RoutePermissionGuard.vue";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import projectRoutes, {
  PROJECT_V1_ROUTE_DETAIL,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  pushNotification,
  useProjectByName,
  useProjectIamPolicy,
  useProjectV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DEFAULT_PROJECT_NAME, UNKNOWN_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useBodyLayoutContext } from "./common";

const props = defineProps<{
  projectId: string;
}>();

const router = useRouter();
const recentProjects = useRecentProjects();
const projectStore = useProjectV1Store();
const { remove: removeVisit } = useRecentVisit();

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);

watchEffect(async () => {
  try {
    const project = await projectStore.getOrFetchProjectByName(
      projectName.value,
      false
    );
    recentProjects.setRecentProject(project.name);
  } catch (err) {
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

const initialized = computed(
  () =>
    ready.value && iamReady.value && project.value.name !== UNKNOWN_PROJECT_NAME
);

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_NAME;
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(project.value, "bb.projects.update");
});

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("px-4");
</script>
