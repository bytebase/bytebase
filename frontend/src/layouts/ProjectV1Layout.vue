<template>
  <template v-if="initialized">
    <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
    <div class="px-4 h-full overflow-auto">
      <template v-if="!hideDefaultProject && isDefaultProject">
        <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
          {{ $t("database.unassigned-databases") }}
        </h1>
        <BBAttention class="mb-4" type="info">
          {{ $t("project.overview.info-slot-content") }}
        </BBAttention>
      </template>
      <QuickActionPanel
        v-if="!hideQuickActionPanel"
        :quick-action-list="quickActionList"
        class="mb-4"
      />
      <router-view
        v-if="hasPermission"
        :project-id="projectId"
        :allow-edit="allowEdit"
        v-bind="$attrs"
      />
      <NoPermissionPlaceholder v-else />
    </div>
  </template>
  <div
    v-else
    class="fixed inset-0 bg-white flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import QuickActionPanel from "@/components/QuickActionPanel.vue";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
} from "@/router/dashboard/projectV1";
import { useAppFeature, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { QuickActionType } from "@/types";
import { DEFAULT_PROJECT_NAME, QuickActionProjectPermissionMap } from "@/types";
import { State } from "@/types/proto/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const route = useRoute();
const router = useRouter();
const recentProjects = useRecentProjects();
const hideQuickAction = useAppFeature("bb.feature.console.hide-quick-action");
const hideDefaultProject = useAppFeature("bb.feature.project.hide-default");
const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project, ready: projectReady } = useProjectByName(projectName);
// Prepare database list of the project.
const { ready: databaseListReady } = useDatabaseV1List(projectName);

const initialized = computed(
  () => projectReady.value && databaseListReady.value
);

watchEffect(() => {
  recentProjects.setRecentProject(project.value.name);
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_NAME;
});

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredProjectPermissionList;
  return getPermissionListFunc ? getPermissionListFunc() : [];
});

const hasPermission = computed(() => {
  return requiredPermissions.value.every((permission) =>
    hasProjectPermissionV2(project.value, permission)
  );
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(project.value, "bb.projects.update");
});

const getQuickActionList = (list: QuickActionType[]): QuickActionType[] => {
  return list.filter((action) => {
    if (!QuickActionProjectPermissionMap.has(action)) {
      return false;
    }
    const hasPermission = QuickActionProjectPermissionMap.get(action)?.every(
      (permission) => hasProjectPermissionV2(project.value, permission)
    );
    return hasPermission;
  });
};

const quickActionListForDatabaseGroup = computed((): QuickActionType[] => {
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  return ["quickaction.bb.group.database-group.create"];
});

const quickActionListForDatabase = computed((): QuickActionType[] => {
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  return [
    "quickaction.bb.database.create",
    "quickaction.bb.project.database.transfer",
    "quickaction.bb.issue.grant.request.querier",
    "quickaction.bb.issue.grant.request.exporter",
  ];
});

const quickActionList = computed(() => {
  switch (route.name) {
    case PROJECT_V1_ROUTE_DATABASES:
      return getQuickActionList(quickActionListForDatabase.value);
    case PROJECT_V1_ROUTE_DATABASE_GROUPS:
      return getQuickActionList(quickActionListForDatabaseGroup.value);
  }
  return [];
});

const hideQuickActionPanel = computed(() => {
  return hideQuickAction.value || quickActionList.value.length === 0;
});
</script>
