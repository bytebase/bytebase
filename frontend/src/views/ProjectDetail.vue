<template>
  <div v-if="quickActionList.length > 0" class="flex-1 pb-2">
    <QuickActionPanel :quick-action-list="quickActionList" />
  </div>
  <template v-if="hash === 'issues'">
    <ProjectIssuesPanel id="issues" :project="project" />
  </template>
  <template v-if="hash === 'branches'">
    <ProjectBranchesPanel id="branches" :project-id="project.uid" />
  </template>
  <template v-if="hash === 'databases'">
    <ProjectDatabasesPanel :database-list="databaseV1List" />
  </template>
  <template v-if="hash === 'database-groups'">
    <ProjectDatabaseGroupPanel :project="project" />
  </template>
  <template v-if="hash === 'change-history'">
    <ProjectChangeHistoryPanel
      id="change-history"
      :database-list="databaseV1List"
    />
  </template>
  <template v-if="hash === 'changelists'">
    <ChangelistDashboard :project="project" />
  </template>
  <template v-if="hash === 'sync-schema'">
    <SyncDatabaseSchema :project="project" />
  </template>
  <template v-if="hash === 'slow-query'">
    <ProjectSlowQueryPanel :project="project" />
  </template>
  <template v-if="hash === 'anomalies'">
    <AnomalyCenterDashboard :project="project" :selected-tab="'database'" />
  </template>
  <template v-if="hash === 'activities'">
    <ProjectActivityPanel id="activity" :project="project" />
  </template>
  <template v-if="!isDefaultProject && hash === 'gitops'">
    <ProjectVersionControlPanel
      id="gitops"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'webhook'">
    <ProjectWebhookPanel
      id="webhook"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'members'">
    <ProjectMemberPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
  <template v-if="!isDefaultProject && hash === 'setting'">
    <ProjectSettingPanel
      id="setting"
      :project="project"
      :allow-edit="allowEdit"
    />
  </template>
</template>

<script lang="ts" setup>
import { pull } from "lodash-es";
import { computed } from "vue";
import { useRoute } from "vue-router";
import AnomalyCenterDashboard from "@/components/AnomalyCenter/AnomalyCenterDashboard.vue";
import ChangelistDashboard from "@/components/Changelist/ChangelistDashboard";
import ProjectDatabaseGroupPanel from "@/components/DatabaseGroup/ProjectDatabaseGroupPanel.vue";
import ProjectIssuesPanel from "@/components/Project/ProjectIssuesPanel.vue";
import { ProjectHash } from "@/components/Project/ProjectSidebar.vue";
import ProjectActivityPanel from "@/components/ProjectActivityPanel.vue";
import ProjectBranchesPanel from "@/components/ProjectBranchesPanel.vue";
import ProjectChangeHistoryPanel from "@/components/ProjectChangeHistoryPanel.vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import ProjectMemberPanel from "@/components/ProjectMember/ProjectMemberPanel.vue";
import ProjectSettingPanel from "@/components/ProjectSettingPanel.vue";
import ProjectSlowQueryPanel from "@/components/ProjectSlowQueryPanel.vue";
import ProjectVersionControlPanel from "@/components/ProjectVersionControlPanel.vue";
import ProjectWebhookPanel from "@/components/ProjectWebhookPanel.vue";
import SyncDatabaseSchema from "@/components/SyncDatabaseSchema/index.vue";
import {
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  hasFeature,
} from "@/store";
import { QuickActionType, DEFAULT_PROJECT_V1_NAME, RoleType } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  idFromSlug,
  sortDatabaseV1List,
  isOwnerOfProjectV1,
  hasPermissionInProjectV1,
  getQuickActionList,
} from "@/utils";

const props = defineProps({
  projectWebhookSlug: {
    default: undefined,
    type: String,
  },
  projectSlug: {
    required: true,
    type: String,
  },
  allowEdit: {
    required: true,
    type: Boolean,
  },
});

const route = useRoute();
const projectV1Store = useProjectV1Store();

const hash = computed(() => route.hash.replace(/^#?/, "") as ProjectHash);

const project = computed(() => {
  return projectV1Store.getProjectByUID(String(idFromSlug(props.projectSlug)));
});

const isDefaultProject = computed((): boolean => {
  return project.value.name === DEFAULT_PROJECT_V1_NAME;
});

useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${project.value.name}"`,
  }))
);

const databaseV1List = computed(() => {
  const list = useDatabaseV1Store().databaseListByProject(project.value.name);
  return sortDatabaseV1List(list);
});

const quickActionMapByRole = computed(() => {
  if (project.value.state === State.ACTIVE) {
    const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
      "quickaction.bb.database.schema.update",
      "quickaction.bb.database.data.update",
      "quickaction.bb.database.create",
      "quickaction.bb.project.database.transfer",
      "quickaction.bb.project.database.transfer-out",
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
      DEVELOPER_QUICK_ACTION_LIST.push(
        "quickaction.bb.database.schema.update",
        "quickaction.bb.database.data.update",
        "quickaction.bb.database.create"
      );
    }
    if (
      hasPermissionInProjectV1(
        project.value.iamPolicy,
        currentUserV1.value,
        "bb.permission.project.transfer-database"
      )
    ) {
      DEVELOPER_QUICK_ACTION_LIST.push(
        "quickaction.bb.project.database.transfer",
        "quickaction.bb.project.database.transfer-out"
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
      ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
      ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
      ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
    ]) as Map<RoleType, QuickActionType[]>;
  }

  return new Map<RoleType, QuickActionType[]>();
});

const isDatabaseHash = computed(() => {
  return hash.value === "databases" || hash.value === "database-groups";
});

const quickActionList = computed(() => {
  if (!isDatabaseHash.value) {
    return [];
  }
  return getQuickActionList(quickActionMapByRole.value);
});
</script>
