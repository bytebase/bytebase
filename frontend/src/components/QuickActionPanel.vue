<template>
  <div
    v-if="availableQuickActionList.length"
    class="overflow-hidden grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch"
    v-bind="$attrs"
  >
    <template
      v-for="(quickAction, index) in availableQuickActionList"
      :key="index"
    >
      <NButton @click="quickAction.action">
        <template #icon>
          <component :is="quickAction.icon" class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ quickAction.title }}
        </NEllipsis>
      </NButton>
    </template>
  </div>

  <Drawer
    :auto-focus="true"
    :show="state.quickActionType !== undefined"
    @close="state.quickActionType = undefined"
  >
    <ProjectCreatePanel
      v-if="state.quickActionType === 'quickaction.bb.project.create'"
      @dismiss="state.quickActionType = undefined"
    />
    <InstanceForm
      v-if="state.quickActionType === 'quickaction.bb.instance.create'"
      :drawer="true"
      @dismiss="state.quickActionType = undefined"
    >
      <DrawerContent :title="$t('quick-action.add-instance')">
        <InstanceFormBody />
        <template #footer>
          <InstanceFormButtons />
        </template>
      </DrawerContent>
    </InstanceForm>
    <CreateDatabasePrepPanel
      v-if="state.quickActionType === 'quickaction.bb.database.create'"
      :project-name="project?.name"
      @dismiss="state.quickActionType = undefined"
    />
    <TransferDatabaseForm
      v-if="
        project &&
        state.quickActionType === 'quickaction.bb.project.database.transfer'
      "
      :project-name="project.name"
      @dismiss="state.quickActionType = undefined"
    />
  </Drawer>

  <template v-if="project">
    <DatabaseGroupPanel
      :show="
        state.quickActionType === 'quickaction.bb.group.database-group.create'
      "
      :project="project"
      @close="state.quickActionType = undefined"
      @created="onDatabaseGroupCreated"
    />
    <GrantRequestPanel
      v-if="
        state.quickActionType ===
          'quickaction.bb.issue.grant.request.querier' ||
        state.quickActionType === 'quickaction.bb.issue.grant.request.exporter'
      "
      :project-name="project.name"
      :role="
        state.quickActionType === 'quickaction.bb.issue.grant.request.querier'
          ? PresetRoleType.PROJECT_QUERIER
          : PresetRoleType.PROJECT_EXPORTER
      "
      @close="state.quickActionType = undefined"
    />
  </template>

  <FeatureModal
    :open="!!state.feature"
    :feature="state.feature"
    @cancel="state.feature = undefined"
  />
</template>

<script lang="ts" setup>
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import {
  PlusIcon,
  DatabaseIcon,
  ListOrderedIcon,
  GalleryHorizontalEndIcon,
  ChevronsDownIcon,
  FileSearchIcon,
  FileDownIcon,
  ShieldCheckIcon,
} from "lucide-vue-next";
import { NButton, NEllipsis } from "naive-ui";
import type { PropType, VNode } from "vue";
import { reactive, computed, watch, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import GrantRequestPanel from "@/components/GrantRequestPanel";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { TransferDatabaseForm } from "@/components/TransferDatabaseForm";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
  PROJECT_V1_ROUTE_MASKING_ACCESS_CREATE,
} from "@/router/dashboard/projectV1";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  useCommandStore,
  useSubscriptionV1Store,
  useProjectV1Store,
  useInstanceResourceList,
  useAppFeature,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  type QuickActionType,
  type DatabaseGroupQuickActionType,
  type FeatureType,
  PresetRoleType,
} from "@/types";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { hasProjectPermissionV2 } from "@/utils";
import DatabaseGroupPanel from "./DatabaseGroup/DatabaseGroupPanel.vue";
import { FeatureModal } from "./FeatureGuard";

interface LocalState {
  feature?: FeatureType;
  quickActionType: QuickActionType | undefined;
}

interface QuickAction {
  type: QuickActionType;
  title: string;
  icon?: VNode;
  hide?: boolean;
  action: () => void;
}

defineOptions({
  inheritAttrs: false,
});

const props = defineProps({
  quickActionList: {
    required: true,
    type: Object as PropType<QuickActionType[]>,
  },
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const commandStore = useCommandStore();
const subscriptionStore = useSubscriptionV1Store();
const projectStore = useProjectV1Store();
const databaseChangeMode = useAppFeature("bb.feature.database-change-mode");

const hasDBAWorkflowFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.dba-workflow");
});

const hasSensitiveDataFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.sensitive-data");
});

const state = reactive<LocalState>({
  quickActionType: undefined,
});

const projectId = computed((): string | undefined => {
  if (route.name?.toString().startsWith(PROJECT_V1_ROUTE_DASHBOARD)) {
    return route.params.projectId as string;
  }
  return undefined;
});

const project = computed(() => {
  if (!projectId.value) {
    return;
  }
  return projectStore.getProjectByName(
    `${projectNamePrefix}${projectId.value}`
  );
});

watch(route, () => {
  state.quickActionType = undefined;
});

const createProject = () => {
  state.quickActionType = "quickaction.bb.project.create";
};

const transferDatabase = () => {
  state.quickActionType = "quickaction.bb.project.database.transfer";
};

const createInstance = () => {
  const instanceList = useInstanceResourceList();
  if (subscriptionStore.instanceCountLimit <= instanceList.value.length) {
    state.feature = "bb.feature.instance-count";
    return;
  }
  state.quickActionType = "quickaction.bb.instance.create";
};

const createDatabase = () => {
  state.quickActionType = "quickaction.bb.database.create";
};

const createEnvironment = () => {
  commandStore.dispatchCommand("bb.environment.create");
};

const reorderEnvironment = () => {
  commandStore.dispatchCommand("bb.environment.reorder");
};

const openDatabaseGroupDrawer = (quickAction: DatabaseGroupQuickActionType) => {
  if (!subscriptionStore.hasFeature("bb.feature.database-grouping")) {
    state.feature = "bb.feature.database-grouping";
    return;
  }
  state.quickActionType = quickAction;
};

const onDatabaseGroupCreated = (databaseGroupName: string) => {
  router.push({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      databaseGroupName,
    },
  });
};

const availableQuickActionList = computed((): QuickAction[] => {
  const fullList: QuickAction[] = [
    {
      type: "quickaction.bb.instance.create",
      title: t("quick-action.add-instance"),
      action: createInstance,
      icon: h(PlusIcon),
    },
    {
      type: "quickaction.bb.database.create",
      title: t("quick-action.new-db"),
      hide:
        databaseChangeMode.value === DatabaseChangeMode.EDITOR ||
        !(
          hasProjectPermissionV2(project.value, "bb.issues.create") &&
          hasProjectPermissionV2(project.value, "bb.plans.create")
        ),
      action: createDatabase,
      icon: h(DatabaseIcon),
    },
    {
      type: "quickaction.bb.group.database-group.create",
      title: t("database-group.create"),
      action: () =>
        openDatabaseGroupDrawer("quickaction.bb.group.database-group.create"),
      icon: h(PlusIcon),
    },
    {
      type: "quickaction.bb.environment.create",
      title: t("environment.create"),
      action: createEnvironment,
      icon: h(PlusIcon),
    },
    {
      type: "quickaction.bb.environment.reorder",
      title: t("common.reorder"),
      action: reorderEnvironment,
      icon: h(ListOrderedIcon),
    },
    {
      type: "quickaction.bb.project.create",
      title: t("quick-action.new-project"),
      action: createProject,
      icon: h(GalleryHorizontalEndIcon),
    },
    {
      type: "quickaction.bb.project.database.transfer",
      title: t("quick-action.transfer-in-db"),
      action: transferDatabase,
      icon: h(ChevronsDownIcon),
    },
    {
      type: "quickaction.bb.issue.grant.request.querier",
      title: t("custom-approval.risk-rule.risk.namespace.request_query"),
      hide: !hasDBAWorkflowFeature.value,
      action: () =>
        (state.quickActionType = "quickaction.bb.issue.grant.request.querier"),
      icon: h(FileSearchIcon),
    },
    {
      type: "quickaction.bb.issue.grant.request.exporter",
      title: t("custom-approval.risk-rule.risk.namespace.request_export"),
      hide: !hasDBAWorkflowFeature.value,
      action: () =>
        (state.quickActionType = "quickaction.bb.issue.grant.request.exporter"),
      icon: h(FileDownIcon),
    },
    {
      type: "quickaction.bb.database.masking-access",
      title: t("project.masking-access.grant-access"),
      hide: !hasSensitiveDataFeature.value,
      action: () => {
        router.push({
          name: PROJECT_V1_ROUTE_MASKING_ACCESS_CREATE,
        });
      },
      icon: h(ShieldCheckIcon),
    },
  ];

  return props.quickActionList.reduce((list, quickAction) => {
    const filter = fullList.filter(
      (item) => item.type === quickAction && !item.hide
    );
    list.push(...filter);
    return list;
  }, [] as QuickAction[]);
});

const kbarActions = computed(() => {
  return availableQuickActionList.value.map((item) => {
    // a QuickActionType starts with "quickaction.bb."
    // it's already namespaced so we don't need prefix here
    // just re-order the identifier to match other kbar action ids' format
    // here `id` looks like "bb.quickaction.instance.create"
    const id = item.type.replace(
      /^quickaction\.bb\.(.+)$/,
      "bb.quickaction.$1"
    );
    return defineAction({
      id,
      section: t("common.quick-action"),
      keywords: "quick action",
      name: item.title,
      perform: item.action,
    });
  });
});

useRegisterActions(kbarActions, true);
</script>
