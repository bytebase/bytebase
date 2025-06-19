<template>
  <template v-if="initialized">
    <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
    <template v-if="isDefaultProject">
      <h1 class="mb-4 text-xl font-bold leading-6 text-main truncate">
        {{ $t("database.unassigned-databases") }}
      </h1>
      <BBAttention class="mb-4" type="info">
        {{ $t("project.overview.info-slot-content") }}
      </BBAttention>
    </template>

    <div
      v-if="!hideQuickActionPanel"
      class="overflow-hidden grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch mb-4"
    >
      <NButton
        v-for="(quickAction, index) in quickActionList"
        :key="index"
        :disabled="quickAction.disabled"
        @click="quickAction.action"
      >
        <template #icon>
          <component :is="quickAction.icon" class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ quickAction.title }}
        </NEllipsis>
      </NButton>
    </div>

    <router-view
      v-if="hasPermission"
      :project-id="projectId"
      :allow-edit="allowEdit"
      v-bind="$attrs"
    />
    <NoPermissionPlaceholder v-else class="py-6">
      <template v-if="hasCreateIssuePermission" #extra>
        <NButton type="primary" @click="state.showRequestRolePanel = true">
          {{ $t("issue.title.request-role") }}
        </NButton>
      </template>
    </NoPermissionPlaceholder>
  </template>
  <div
    v-else
    class="fixed inset-0 bg-white flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>

  <GrantRequestPanel
    v-if="state.showRequestRolePanel"
    :project-name="project.name"
    @close="state.showRequestRolePanel = false"
  />

  <IAMRemindModal :project-name="project.name" />
</template>

<script lang="tsx" setup>
import { UsersIcon } from "lucide-vue-next";
import { NButton, NEllipsis, NSpin } from "naive-ui";
import type { ClientError } from "nice-grpc-web";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { FeatureBadge } from "@/components/FeatureGuard";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import IAMRemindModal from "@/components/IAMRemindModal.vue";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_MEMBERS,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  pushNotification,
  usePermissionStore,
  useProjectByName,
  useProjectV1Store,
  featureToRef,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  DEFAULT_PROJECT_NAME,
  PresetRoleType,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useBodyLayoutContext } from "./common";

interface LocalState {
  showRequestRolePanel: boolean;
}

const props = defineProps<{
  projectId: string;
}>();

const state = reactive<LocalState>({
  showRequestRolePanel: false,
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const recentProjects = useRecentProjects();
const projectStore = useProjectV1Store();
const { remove: removeVisit } = useRecentVisit();
const permissionStore = usePermissionStore();

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);

watchEffect(async () => {
  try {
    const project = await projectStore.getOrFetchProjectByName(
      projectName.value
    );
    recentProjects.setRecentProject(project.name);
  } catch (err) {
    console.error(err);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to fetch project ${props.projectId}`,
      description: (err as ClientError).details,
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

const initialized = computed(
  () => ready && project.value.name !== UNKNOWN_PROJECT_NAME
);

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_NAME;
});

const requiredPermissions = computed(() => {
  const getPermissionListFunc =
    router.currentRoute.value.meta.requiredPermissionList;
  const permissions = getPermissionListFunc ? getPermissionListFunc() : [];
  permissions.push("bb.projects.get");
  return permissions;
});

const hasCreateIssuePermission = computed(() =>
  hasProjectPermissionV2(project.value, "bb.issues.create")
);

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

const isProjectOwner = computed(() => {
  const roles = permissionStore.currentRoleListInProjectV1(project.value);
  return roles.includes(PresetRoleType.PROJECT_OWNER);
});

const hasRequestRoleFeature = featureToRef(
  PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW
);

const quickActionListForDatabase = computed(() => {
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  const actions = [];

  if (
    !isProjectOwner.value &&
    hasProjectPermissionV2(project.value, "bb.issues.create")
  ) {
    actions.push({
      title: t("issue.title.request-role"),
      disabled: !hasRequestRoleFeature.value,
      icon: () =>
        hasRequestRoleFeature.value ? (
          <UsersIcon />
        ) : (
          <FeatureBadge feature={PlanFeature.FEATURE_REQUEST_ROLE_WORKFLOW} />
        ),
      action: () => (state.showRequestRolePanel = true),
    });
  }

  return actions;
});

const quickActionList = computed(() => {
  switch (route.name) {
    case PROJECT_V1_ROUTE_DATABASES:
    case PROJECT_V1_ROUTE_MEMBERS:
      return quickActionListForDatabase.value;
  }
  return [];
});

const hideQuickActionPanel = computed(() => {
  return quickActionList.value.length === 0;
});

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("px-4");
</script>
