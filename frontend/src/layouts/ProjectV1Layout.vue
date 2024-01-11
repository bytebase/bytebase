<template>
  <ArchiveBanner v-if="project.state === State.DELETED" class="py-2" />
  <div class="p-4 h-full overflow-auto">
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
    <router-view :project-id="projectId" :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { pull } from "lodash-es";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import HideInStandaloneMode from "@/components/misc/HideInStandaloneMode.vue";
import {
  PROJECT_V1_DATABASES,
  PROJECT_V1_DATABASE_GROUPS,
} from "@/router/dashboard/projectV1";
import {
  useProjectV1Store,
  useCurrentUserV1,
  useActivityV1Store,
  hasFeature,
  usePageMode,
  pushNotification,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  DEFAULT_PROJECT_V1_NAME,
  QuickActionType,
  RoleType,
  activityName,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  isOwnerOfProjectV1,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
  getQuickActionList,
} from "@/utils";

const props = defineProps({
  projectId: {
    required: true,
    type: String,
  },
});

const route = useRoute();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const activityV1Store = useActivityV1Store();
const pageMode = usePageMode();
const { t } = useI18n();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

const allowEdit = computed(() => {
  if (project.value.state === State.DELETED) {
    return false;
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-project",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (
    hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.manage-general"
    )
  ) {
    return true;
  }
  return false;
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
    !hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      currentUserV1.value.userRole
    ) &&
    !hasPermissionInProjectV1(
      project.value.iamPolicy,
      currentUserV1.value,
      "bb.permission.project.change-database"
    )
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

const quickActionForDatabaseGroup = computed((): QuickActionType[] => {
  if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
    return [];
  }
  return [
    "quickaction.bb.database.schema.update",
    "quickaction.bb.database.data.update",
    "quickaction.bb.group.database-group.create",
    "quickaction.bb.group.table-group.create",
  ];
});

const quickActionMapByRole = computed(() => {
  if (project.value.state === State.ACTIVE) {
    const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
      "quickaction.bb.database.create",
      "quickaction.bb.project.database.transfer",
    ];
    const DEVELOPER_QUICK_ACTION_LIST: QuickActionType[] = [];

    const currentUserV1 = useCurrentUserV1();
    if (
      project.value.name !== DEFAULT_PROJECT_V1_NAME &&
      hasPermissionInProjectV1(
        project.value.iamPolicy,
        currentUserV1.value,
        "bb.permission.project.change-database"
      )
    ) {
      // Default project (Unassigned databases) are not allowed
      // to be changed.
      DEVELOPER_QUICK_ACTION_LIST.push("quickaction.bb.database.create");
    }
    if (
      hasPermissionInProjectV1(
        project.value.iamPolicy,
        currentUserV1.value,
        "bb.permission.project.transfer-database"
      )
    ) {
      DEVELOPER_QUICK_ACTION_LIST.push(
        "quickaction.bb.project.database.transfer"
      );
    }
    if (!isOwnerOfProjectV1(project.value.iamPolicy, currentUserV1.value)) {
      DEVELOPER_QUICK_ACTION_LIST.push(
        "quickaction.bb.issue.grant.request.querier",
        "quickaction.bb.issue.grant.request.exporter"
      );
    }

    if (hasFeature("bb.feature.dba-workflow")) {
      pull(DEVELOPER_QUICK_ACTION_LIST, "quickaction.bb.database.create");
    }

    return new Map([
      ["OWNER", [...DBA_AND_OWNER_QUICK_ACTION_LIST]],
      ["DBA", [...DBA_AND_OWNER_QUICK_ACTION_LIST]],
      ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
    ]) as Map<RoleType, QuickActionType[]>;
  }

  return new Map<RoleType, QuickActionType[]>();
});

const isDatabaseHash = computed(() => {
  return (
    route.name === PROJECT_V1_DATABASES ||
    route.name === PROJECT_V1_DATABASE_GROUPS
  );
});

const quickActionList = computed(() => {
  if (!isDatabaseHash.value) {
    return [];
  }
  if (route.name === PROJECT_V1_DATABASE_GROUPS) {
    return quickActionForDatabaseGroup.value;
  }
  return getQuickActionList(quickActionMapByRole.value);
});

const showQuickActionPanel = computed(() => {
  return pageMode.value === "BUNDLED" && quickActionList.value.length > 0;
});
</script>
