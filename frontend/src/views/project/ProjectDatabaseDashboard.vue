<template>
  <div v-if="showQuickActionPanel" class="flex-1 pb-4">
    <QuickActionPanel :quick-action-list="quickActionList" />
  </div>
  <ProjectDatabasesPanel :database-list="databaseV1List" />
</template>

<script lang="ts" setup>
import { pull } from "lodash-es";
import { computed } from "vue";
import ProjectDatabasesPanel from "@/components/ProjectDatabasesPanel.vue";
import {
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useProjectV1Store,
  useCurrentUserV1,
  hasFeature,
  usePageMode,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { QuickActionType, DEFAULT_PROJECT_V1_NAME, RoleType } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  sortDatabaseV1List,
  isOwnerOfProjectV1,
  hasPermissionInProjectV1,
  getQuickActionList,
  isDatabaseV1Alterable,
} from "@/utils";

const props = defineProps({
  projectId: {
    required: true,
    type: String,
  },
});

const currentUser = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const pageMode = usePageMode();

const project = computed(() => {
  return projectV1Store.getProjectByName(
    `${projectNamePrefix}${props.projectId}`
  );
});

useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${project.value.name}"`,
  }))
);

const databaseV1List = computed(() => {
  let list = useDatabaseV1Store().databaseListByProject(project.value.name);
  list = sortDatabaseV1List(list);
  // In standalone mode, only show alterable databases.
  if (pageMode.value === "STANDALONE") {
    list = list.filter((db) => isDatabaseV1Alterable(db, currentUser.value));
  }
  return list;
});

const quickActionMapByRole = computed(() => {
  if (project.value.state === State.ACTIVE) {
    const DBA_AND_OWNER_QUICK_ACTION_LIST: QuickActionType[] = [
      "quickaction.bb.database.schema.update",
      "quickaction.bb.database.data.update",
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
      ["OWNER", DBA_AND_OWNER_QUICK_ACTION_LIST],
      ["DBA", DBA_AND_OWNER_QUICK_ACTION_LIST],
      ["DEVELOPER", DEVELOPER_QUICK_ACTION_LIST],
    ]) as Map<RoleType, QuickActionType[]>;
  }

  return new Map<RoleType, QuickActionType[]>();
});

const quickActionList = computed(() => {
  return getQuickActionList(quickActionMapByRole.value);
});

const showQuickActionPanel = computed(() => {
  return pageMode.value === "BUNDLED" && quickActionList.value.length > 0;
});
</script>
