<template>
  <div
    class="pt-1 overflow-hidden grid grid-cols-5 gap-x-2 gap-y-4 md:inline-flex items-stretch"
  >
    <template v-for="(quickAction, index) in quickActionList" :key="index">
      <div
        v-if="quickAction === 'quickaction.bb.instance.create'"
        class="flex flex-col items-center w-24"
        data-label="bb-quick-action-add-instance"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createInstance">
          <heroicons-outline:plus-sm class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.add-instance") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.user.manage'"
        class="flex flex-col items-center w-24"
      >
        <router-link to="/setting/member" class="btn-icon-primary p-3">
          <heroicons-outline:users class="w-5 h-5" />
        </router-link>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.manage-user") }}
        </h3>
      </div>

      <template v-if="shouldShowAlterDatabaseEntries">
        <div
          v-if="quickAction === 'quickaction.bb.database.create'"
          class="flex flex-col items-center w-24"
          data-label="bb-quick-action-new-db"
        >
          <button class="btn-icon-primary p-3" @click.prevent="createDatabase">
            <heroicons-outline:database class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("quick-action.new-db") }}
          </h3>
        </div>

        <div
          v-if="quickAction === 'quickaction.bb.database.schema.update'"
          class="flex flex-col items-center w-24"
        >
          <button
            class="btn-icon-primary p-3"
            data-label="bb-alter-schema-button"
            @click.prevent="alterSchema"
          >
            <heroicons-outline:pencil-alt class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("database.alter-schema") }}
          </h3>
        </div>

        <div
          v-if="quickAction === 'quickaction.bb.database.data.update'"
          class="flex flex-col items-center w-24"
        >
          <button class="btn-icon-primary p-3" @click.prevent="changeData">
            <heroicons-outline:pencil class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("database.change-data") }}
          </h3>
        </div>

        <div
          v-if="quickAction === 'quickaction.bb.database.schema.design'"
          class="flex flex-col items-center w-24"
        >
          <button class="btn-icon-primary p-3" @click.prevent="designSchema">
            <heroicons-outline:table-cells class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("schema-designer.quick-action") }}
          </h3>
        </div>
      </template>

      <div
        v-if="isDev() && quickAction === 'quickaction.bb.database.troubleshoot'"
        class="flex flex-col items-center w-24"
      >
        <router-link to="/issue/new" class="btn-icon-primary p-3">
          <heroicons-outline:hand class="w-5 h-5" />
        </router-link>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.troubleshoot") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.environment.create'"
        class="flex flex-col items-center w-36"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createEnvironment">
          <heroicons-outline:plus-sm class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("environment.create") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.environment.reorder'"
        class="flex flex-col items-center w-24"
      >
        <button
          class="btn-icon-primary p-3"
          @click.prevent="reorderEnvironment"
        >
          <heroicons-outline:selector class="transform rotate-90 w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("common.reorder") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.project.create'"
        class="flex flex-col items-center w-24"
        data-label="bb-quick-action-new-project"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createProject">
          <heroicons-outline:template class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.new-project") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.project.database.transfer'"
        class="flex flex-col items-center w-24"
      >
        <button class="btn-icon-primary p-3" @click.prevent="transferDatabase">
          <heroicons-outline:chevron-double-down class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.transfer-in-db") }}
        </h3>
      </div>

      <div
        v-if="quickAction === 'quickaction.bb.project.database.transfer'"
        class="flex flex-col items-center w-24"
      >
        <button
          class="btn-icon-primary p-3"
          @click.prevent="transferOutDatabase"
        >
          <heroicons-outline:chevron-double-up class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.transfer-out-db") }}
        </h3>
      </div>

      <template v-if="hasDBAWorkflowFeature">
        <div
          v-if="quickAction === 'quickaction.bb.issue.grant.request.querier'"
          class="flex flex-col items-center w-24"
        >
          <button
            class="btn-icon-primary p-3"
            @click.prevent="state.showRequestQueryPanel = true"
          >
            <heroicons-outline:document-search class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("quick-action.request-query") }}
          </h3>
        </div>

        <div
          v-if="quickAction === 'quickaction.bb.issue.grant.request.exporter'"
          class="flex flex-col items-center w-24"
        >
          <button
            class="btn-icon-primary p-3"
            @click.prevent="state.showRequestExportPanel = true"
          >
            <heroicons-outline:document-download class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("quick-action.request-export") }}
          </h3>
        </div>
      </template>

      <div
        v-if="quickAction === 'quickaction.bb.subscription.license-assignment'"
        class="flex flex-col items-center w-24"
      >
        <button
          class="btn-icon-primary p-3"
          @click.prevent="
            () =>
              (state.quickActionType =
                'quickaction.bb.subscription.license-assignment')
          "
        >
          <heroicons-outline:academic-cap class="w-5 h-5" />
        </button>
        <h3
          class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
        >
          {{ $t("subscription.instance-assignment.assign-license") }}
        </h3>
      </div>
    </template>
  </div>

  <Drawer
    :auto-focus="true"
    :trap-focus="false"
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
    <DesignSchemaPrepForm
      v-if="state.quickActionType === 'quickaction.bb.database.schema.design'"
      :project-id="projectId"
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
    <TransferOutDatabaseForm
      v-if="
        projectId &&
        state.quickActionType === 'quickaction.bb.project.database.transfer-out'
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
    @close="state.showRequestExportPanel = false"
  />

  <InstanceAssignment
    :show="
      state.quickActionType === 'quickaction.bb.subscription.license-assignment'
    "
    @dismiss="state.quickActionType = undefined"
  />

  <FeatureModal
    :open="state.showFeatureModal && state.feature"
    :feature="state.feature"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { reactive, PropType, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";

import { QuickActionType } from "@/types";
import { idFromSlug, isDev } from "@/utils";
import {
  useInstanceV1Store,
  useCommandStore,
  useCurrentUserIamPolicy,
  useProjectV1ListByCurrentUser,
  useSubscriptionV1Store,
} from "@/store";
import { Drawer } from "@/components/v2";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import InstanceForm from "@/components/InstanceForm/";
import AlterSchemaPrepForm from "@/components/AlterSchemaPrepForm/";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import TransferDatabaseForm from "@/components/TransferDatabaseForm.vue";
import TransferOutDatabaseForm from "@/components/TransferOutDatabaseForm";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import RequestQueryPanel from "@/components/Issue/panel/RequestQueryPanel/index.vue";
import DesignSchemaPrepForm from "@/components/SchemaDesigner/PrepForm/index.vue";

interface LocalState {
  feature?: string;
  showFeatureModal: boolean;
  quickActionType: QuickActionType | undefined;
  showRequestQueryPanel: boolean;
  showRequestExportPanel: boolean;
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

const transferOutDatabase = () => {
  state.quickActionType = "quickaction.bb.project.database.transfer-out";
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

const designSchema = () => {
  state.quickActionType = "quickaction.bb.database.schema.design";
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

const QuickActionMap: Record<string, Partial<Action>> = {
  "quickaction.bb.instance.create": {
    name: t("quick-action.add-instance"),
    perform: () => createInstance(),
  },
  "quickaction.bb.user.manage": {
    name: t("quick-action.manage-user"),
    perform: () => router.push({ name: "setting.workspace.member" }),
  },
  "quickaction.bb.database.create": {
    name: t("quick-action.new-db"),
    perform: () => createDatabase(),
  },
  "quickaction.bb.database.schema.update": {
    name: t("database.alter-schema"),
    perform: () => alterSchema(),
  },
  "quickaction.bb.database.troubleshoot": {
    name: t("quick-action.troubleshoot"),
    perform: () => router.push({ path: "/issue/new" }),
  },
  "quickaction.bb.environment.create": {
    name: t("quick-action.add-environment"),
    perform: () => createEnvironment(),
  },
  "quickaction.bb.environment.reorder": {
    name: t("common.reorder"),
    perform: () => reorderEnvironment(),
  },
  "quickaction.bb.project.create": {
    name: t("quick-action.new-project"),
    perform: () => createProject(),
  },
  "quickaction.bb.project.database.transfer": {
    name: t("quick-action.transfer-in-db"),
    perform: () => transferDatabase(),
  },
  "quickaction.bb.subscription.license-assignment": {
    name: t("subscription.instance-assignment.manage-license"),
    perform: () =>
      (state.quickActionType =
        "quickaction.bb.subscription.license-assignment"),
  },
};

const kbarActions = computed(() => {
  return props.quickActionList
    .filter((qa) => qa in QuickActionMap)
    .map((qa) => {
      // a QuickActionType starts with "quickaction.bb."
      // it's already namespaced so we don't need prefix here
      // just re-order the identifier to match other kbar action ids' format
      // here `id` looks like "bb.quickaction.instance.create"
      const id = qa.replace(/^quickaction\.bb\.(.+)$/, "bb.quickaction.$1");
      return defineAction({
        id,
        section: t("common.quick-action"),
        keywords: "quick action",
        ...QuickActionMap[qa],
      });
    });
});

useRegisterActions(kbarActions, true);
</script>
