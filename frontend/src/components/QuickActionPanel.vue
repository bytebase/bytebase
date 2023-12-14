<template>
  <div
    class="pt-1 overflow-hidden grid grid-cols-3 gap-x-2 gap-y-4 md:inline-flex items-stretch"
  >
    <template
      v-for="(quickAction, index) in availableQuickActionList"
      :key="index"
    >
      <NButton @click="quickAction.action">
        <template #icon>
          <component :is="quickAction.icon" class="h-4 w-4" />
        </template>
        {{ quickAction.title }}
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
    />
    <CreateDatabasePrepPanel
      v-if="state.quickActionType === 'quickaction.bb.database.create'"
      :project-id="projectId"
      @dismiss="state.quickActionType = undefined"
    />
    <AlterSchemaPrepForm
      v-if="state.quickActionType === 'quickaction.bb.database.schema.update'"
      :project-id="projectId"
      :type="'bb.issue.database.schema.update'"
      @dismiss="state.quickActionType = undefined"
    />
    <AlterSchemaPrepForm
      v-if="state.quickActionType === 'quickaction.bb.database.data.update'"
      :project-id="projectId"
      :type="'bb.issue.database.data.update'"
      @dismiss="state.quickActionType = undefined"
    />
    <TransferDatabaseForm
      v-if="
        projectId &&
        state.quickActionType === 'quickaction.bb.project.database.transfer'
      "
      :project-id="projectId"
      @dismiss="state.quickActionType = undefined"
    />
  </Drawer>

  <RequestQueryPanel
    v-if="state.showRequestQueryPanel"
    @close="state.showRequestQueryPanel = false"
  />

  <RequestExportPanel
    v-if="state.showRequestExportPanel"
    :redirect-to-issue-page="true"
    @close="state.showRequestExportPanel = false"
  />

  <FeatureModal
    :open="state.showFeatureModal && !!state.feature"
    :feature="(state.feature as FeatureType)"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import {
  PlusIcon,
  DatabaseIcon,
  PencilIcon,
  PenSquareIcon,
  ListOrderedIcon,
  GalleryHorizontalEndIcon,
  ChevronsDownIcon,
  FileSearchIcon,
  FileDownIcon,
} from "lucide-vue-next";
import { reactive, PropType, computed, watch, VNode, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import AlterSchemaPrepForm from "@/components/AlterSchemaPrepForm/";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import InstanceForm from "@/components/InstanceForm/";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import RequestQueryPanel from "@/components/Issue/panel/RequestQueryPanel/index.vue";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import TransferDatabaseForm from "@/components/TransferDatabaseForm.vue";
import { Drawer } from "@/components/v2";
import {
  useInstanceV1Store,
  useCommandStore,
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
  useSubscriptionV1Store,
} from "@/store";
import { QuickActionType, FeatureType } from "@/types";
import { idFromSlug } from "@/utils";

interface LocalState {
  feature?: string;
  showFeatureModal: boolean;
  quickActionType: QuickActionType | undefined;
  showRequestQueryPanel: boolean;
  showRequestExportPanel: boolean;
}

interface QuickAction {
  type: QuickActionType;
  title: string;
  icon?: VNode;
  hide?: boolean;
  action: () => void;
}

const props = defineProps({
  quickActionList: {
    required: true,
    type: Object as PropType<QuickActionType[]>,
  },
});

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const commandStore = useCommandStore();
const subscriptionStore = useSubscriptionV1Store();

const hasDBAWorkflowFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.dba-workflow");
});

const state = reactive<LocalState>({
  showFeatureModal: false,
  quickActionType: undefined,
  showRequestQueryPanel: false,
  showRequestExportPanel: false,
});

const projectId = computed((): string | undefined => {
  if (router.currentRoute.value.name == "workspace.project.detail") {
    const parts = router.currentRoute.value.path.split("/");
    return String(idFromSlug(parts[parts.length - 1]));
  }
  return undefined;
});

// Only show alter schema and change data if the user has permission to alter schema of at least one project.
const shouldShowAlterDatabaseEntries = computed(() => {
  const { projectList } = useProjectV1ListByCurrentUser();
  const currentUserIamPolicy = useCurrentUserIamPolicy();
  return projectList.value
    .map((project) => {
      return currentUserIamPolicy.allowToChangeDatabaseOfProject(project.name);
    })
    .includes(true);
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
  const instanceList = useInstanceV1Store().activeInstanceList;
  if (subscriptionStore.instanceCountLimit <= instanceList.length) {
    state.feature = "bb.feature.instance-count";
    state.showFeatureModal = true;
    return;
  }
  state.quickActionType = "quickaction.bb.instance.create";
};

const alterSchema = () => {
  state.quickActionType = "quickaction.bb.database.schema.update";
};

const changeData = () => {
  state.quickActionType = "quickaction.bb.database.data.update";
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
      hide: !shouldShowAlterDatabaseEntries.value,
      action: createDatabase,
      icon: h(DatabaseIcon),
    },
    {
      type: "quickaction.bb.database.schema.update",
      title: t("database.edit-schema"),
      hide: !shouldShowAlterDatabaseEntries.value,
      action: alterSchema,
      icon: h(PenSquareIcon),
    },
    {
      type: "quickaction.bb.database.data.update",
      title: t("database.change-data"),
      hide: !shouldShowAlterDatabaseEntries.value,
      action: changeData,
      icon: h(PencilIcon),
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
      title: t("quick-action.request-query-permission"),
      hide: !hasDBAWorkflowFeature.value,
      action: () => (state.showRequestQueryPanel = true),
      icon: h(FileSearchIcon),
    },
    {
      type: "quickaction.bb.issue.grant.request.exporter",
      title: t("quick-action.request-export-permission"),
      hide: !hasDBAWorkflowFeature.value,
      action: () => (state.showRequestExportPanel = true),
      icon: h(FileDownIcon),
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
  return (
    availableQuickActionList.value
      // .filter((qa) => qa in QuickActionMap)
      .map((item) => {
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
      })
  );
});

useRegisterActions(kbarActions, true);
</script>
