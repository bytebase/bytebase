<template>
  <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
  <div class="px-4 h-full overflow-auto">
    <HideInStandaloneMode>
      <template v-if="isDefaultProject">
        <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
          {{ $t("database.unassigned-databases") }}
        </h1>
      </template>
      <BBAttention
        v-if="isDefaultProject"
        class="mb-4"
        type="info"
        :title="$t('project.overview.info-slot-content')"
      />
    </HideInStandaloneMode>
    <QuickActionPanel
      v-if="showQuickActionPanel"
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

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { computed, onMounted, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import HideInStandaloneMode from "@/components/misc/HideInStandaloneMode.vue";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG,
} from "@/router/dashboard/projectV1";
import {
  useProjectV1Store,
  useCurrentUserV1,
  useActivityV1Store,
  usePageMode,
  pushNotification,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  ProjectPermission,
  DEFAULT_PROJECT_V1_NAME,
  QuickActionType,
  activityName,
  QuickActionProjectPermissionMap,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const route = useRoute();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const activityV1Store = useActivityV1Store();
const pageMode = usePageMode();
const { t } = useI18n();
const recentProjects = useRecentProjects();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

watchEffect(() => {
  recentProjects.setRecentProject(project.value.name);
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const currentUser = useCurrentUserV1();

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredProjectPermissionList;
  return getPermissionListFunc ? getPermissionListFunc() : [];
});

const hasPermission = computed(() => {
  return requiredPermissions.value.every((permission) =>
    hasProjectPermissionV2(project.value, currentUser.value, permission)
  );
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  return hasProjectPermissionV2(
    project.value,
    currentUserV1.value,
    "bb.projects.update"
  );
});

const cachedNotifiedActivities = useLocalStorage<string[]>(
  `bb.project.${props.projectId}.activities`,
  []
);

const maximumCachedActivities = 5;

onMounted(async () => {
  await projectV1Store.getOrFetchProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );

  if (
    !hasProjectPermissionV2(project.value, currentUserV1.value, "bb.issues.get")
  ) {
    return;
  }
  activityV1Store
    .fetchActivityList({
      pageSize: 1,
      order: "desc",
      action: [LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH],
      resource: project.value.name,
    })
    .then((resp) => {
      for (const activity of resp.logEntities) {
        if (cachedNotifiedActivities.value.includes(activity.name)) {
          continue;
        }
        cachedNotifiedActivities.value.push(activity.name);
        if (cachedNotifiedActivities.value.length > maximumCachedActivities) {
          cachedNotifiedActivities.value.shift();
        }

        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: activityName(activity.action),
          manualHide: true,
          link: `/${project.value.name}/activities`,
          linkTitle: t("common.view"),
        });
        break;
      }
    });
});

// Only show the following actions if users don't have required permissions.
const quickActionWithoutPermission: Map<QuickActionType, ProjectPermission> =
  new Map([
    ["quickaction.bb.issue.grant.request.querier", "bb.databases.query"],
    ["quickaction.bb.issue.grant.request.exporter", "bb.databases.export"],
  ]);

const getQuickActionList = (list: QuickActionType[]): QuickActionType[] => {
  return list.filter((action) => {
    if (!QuickActionProjectPermissionMap.has(action)) {
      return false;
    }
    const hasPermission = QuickActionProjectPermissionMap.get(action)?.every(
      (permission) =>
        hasProjectPermissionV2(project.value, currentUserV1.value, permission)
    );
    if (quickActionWithoutPermission.has(action)) {
      if (
        hasProjectPermissionV2(
          project.value,
          currentUserV1.value,
          quickActionWithoutPermission.get(action)!
        )
      )
        return false;
    }
    return hasPermission;
  });
};

const quickActionListForDatabaseGroup = computed((): QuickActionType[] => {
  if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
    return [];
  }
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  return [
    "quickaction.bb.database.schema.update",
    "quickaction.bb.database.data.update",
    "quickaction.bb.group.database-group.create",
    "quickaction.bb.group.table-group.create",
  ];
});

const quickActionListForDeploymentConfig = computed((): QuickActionType[] => {
  if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
    return [];
  }
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  return [
    "quickaction.bb.database.schema.update",
    "quickaction.bb.database.data.update",
  ];
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
    case PROJECT_V1_ROUTE_DEPLOYMENT_CONFIG:
      return getQuickActionList(quickActionListForDeploymentConfig.value);
  }
  return [];
});

const showQuickActionPanel = computed(() => {
  return pageMode.value === "BUNDLED" && quickActionList.value.length > 0;
});
</script>
