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

      <div
        v-if="!hideQuickActionPanel"
        class="overflow-hidden grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch mb-4"
      >
        <NButton
          v-for="(quickAction, index) in quickActionList"
          :key="index"
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
      <NoPermissionPlaceholder v-else />
    </div>
  </template>
  <div
    v-else
    class="fixed inset-0 bg-white flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>

  <GrantRequestPanel
    v-if="!!state.requestRole"
    :project-name="project.name"
    :role="state.requestRole"
    @close="state.requestRole = undefined"
  />
</template>

<script lang="ts" setup>
import { FileSearchIcon, FileDownIcon } from "lucide-vue-next";
import { NButton, NEllipsis, NSpin } from "naive-ui";
import type { ClientError } from "nice-grpc-web";
import { computed, watchEffect, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import {
  PROJECT_V1_ROUTE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  hasFeature,
  useAppFeature,
  useProjectV1Store,
  pushNotification,
  usePermissionStore,
  usePolicyByParentAndType,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import {
  UNKNOWN_PROJECT_NAME,
  DEFAULT_PROJECT_NAME,
  PresetRoleType,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { hasProjectPermissionV2 } from "@/utils";

interface LocalState {
  requestRole?:
    | PresetRoleType.SQL_EDITOR_USER
    | PresetRoleType.PROJECT_EXPORTER;
}

const props = defineProps<{
  projectId: string;
}>();

const state = reactive<LocalState>({});

const route = useRoute();
const router = useRouter();
const recentProjects = useRecentProjects();
const projectStore = useProjectV1Store();
const { remove: removeVisit } = useRecentVisit();
const permissionStore = usePermissionStore();
const { t } = useI18n();

const hideQuickAction = useAppFeature("bb.feature.console.hide-quick-action");
const hideDefaultProject = useAppFeature("bb.feature.project.hide-default");
const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
// Prepare database list of the project.
const { ready: databaseListReady } = useDatabaseV1List(projectName);

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

const project = computed(() =>
  projectStore.getProjectByName(projectName.value)
);

const exportDataPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_EXPORT,
  }))
);

const initialized = computed(
  () => project.value.name !== UNKNOWN_PROJECT_NAME && databaseListReady.value
);

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

const hasDBAWorkflowFeature = computed(() => {
  return hasFeature("bb.feature.dba-workflow");
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

const quickActionListForDatabase = computed(() => {
  if (project.value.state !== State.ACTIVE) {
    return [];
  }

  const actions = [];

  if (
    !isProjectOwner.value &&
    hasProjectPermissionV2(project.value, "bb.issues.create") &&
    hasDBAWorkflowFeature.value
  ) {
    actions.push({
      title: t("custom-approval.risk-rule.risk.namespace.request_query"),
      icon: () => h(FileSearchIcon),
      action: () => (state.requestRole = PresetRoleType.SQL_EDITOR_USER),
    });

    if (!exportDataPolicy.value?.exportDataPolicy?.disable) {
      actions.push({
        title: t("custom-approval.risk-rule.risk.namespace.request_export"),
        icon: () => h(FileDownIcon),
        action: () => (state.requestRole = PresetRoleType.PROJECT_EXPORTER),
      });
    }
  }

  return actions;
});

const quickActionList = computed(() => {
  switch (route.name) {
    case PROJECT_V1_ROUTE_DATABASES:
      return quickActionListForDatabase.value;
  }
  return [];
});

const hideQuickActionPanel = computed(() => {
  return hideQuickAction.value || quickActionList.value.length === 0;
});
</script>
