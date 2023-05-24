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
          v-if="quickAction === 'quickaction.bb.database.request'"
          class="flex flex-col items-center w-24"
        >
          <button class="btn-icon-primary p-3" @click.prevent="requestDatabase">
            <heroicons-outline:database class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("quick-action.request-db") }}
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

      <template v-if="hasCustomRoleFeature">
        <div
          v-if="quickAction === 'quickaction.bb.issue.grant.request.querier'"
          class="flex flex-col items-center w-24"
        >
          <button
            class="btn-icon-primary p-3"
            @click.prevent="createRequestQueryIssue"
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
            @click.prevent="createExportDataIssue"
          >
            <heroicons-outline:document-download class="w-5 h-5" />
          </button>
          <h3
            class="flex-1 mt-1.5 text-center text-sm font-normal text-main tracking-tight"
          >
            {{ $t("quick-action.export-data") }}
          </h3>
        </div>
      </template>
    </template>
  </div>
  <BBModal
    v-if="state.showModal"
    class="relative overflow-hidden"
    :title="state.modalTitle"
    :subtitle="state.modalSubtitle"
    :data-label="`bb-${kebabCase(state.modalTitle)}-modal`"
    @close="state.showModal = false"
  >
    <template v-if="state.quickActionType == 'quickaction.bb.project.create'">
      <ProjectCreate @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.instance.create'"
    >
      <InstanceForm :modal="true" @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bb.database.schema.update'
      "
    >
      <AlterSchemaPrepForm
        :project-id="projectId"
        :type="'bb.issue.database.schema.update'"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.data.update'"
    >
      <AlterSchemaPrepForm
        :project-id="projectId"
        :type="'bb.issue.database.data.update'"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.create'"
    >
      <CreateDatabasePrepForm
        :project-id="projectId"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.request'"
    >
      <RequestDatabasePrepForm @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bb.project.database.transfer'
      "
    >
      <TransferDatabaseForm
        v-if="projectId"
        :project-id="projectId"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bb.project.database.transfer-out'
      "
    >
      <TransferOutDatabaseForm
        v-if="projectId"
        :project-id="projectId"
        @dismiss="state.showModal = false"
      />
    </template>
  </BBModal>
  <FeatureModal
    v-if="state.showFeatureModal && state.featureName !== ''"
    :feature="state.featureName"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { kebabCase } from "lodash-es";
import { reactive, PropType, computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { QuickActionType } from "../types";
import { idFromSlug, isDev } from "../utils";
import {
  useCommandStore,
  useCurrentUserIamPolicy,
  useInstanceStore,
  useProjectV1ListByCurrentUser,
  useRouterStore,
  useSubscriptionStore,
} from "@/store";
import ProjectCreate from "../components/ProjectCreate.vue";
import InstanceForm from "../components/InstanceForm.vue";
import AlterSchemaPrepForm from "./AlterSchemaPrepForm/";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import RequestDatabasePrepForm from "../components/RequestDatabasePrepForm.vue";
import TransferDatabaseForm from "../components/TransferDatabaseForm.vue";
import TransferOutDatabaseForm from "../components/TransferOutDatabaseForm";

interface LocalState {
  showModal: boolean;
  featureName: string;
  showFeatureModal: boolean;
  modalTitle: string;
  modalSubtitle: string;
  quickActionType: QuickActionType;
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
const routerStore = useRouterStore();
const commandStore = useCommandStore();
const subscriptionStore = useSubscriptionStore();

const hasCustomRoleFeature = computed(() => {
  return subscriptionStore.hasFeature("bb.feature.custom-role");
});

const state = reactive<LocalState>({
  showModal: false,
  featureName: "",
  showFeatureModal: false,
  modalTitle: "",
  modalSubtitle: "",
  quickActionType: "quickaction.bb.instance.create",
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
  state.showModal = false;
});

const createProject = () => {
  state.modalTitle = t("quick-action.create-project");
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.project.create";
  state.showModal = true;
};

const transferDatabase = () => {
  state.modalTitle = t("quick-action.transfer-in-db-title");
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.project.database.transfer";
  state.showModal = true;
};

const transferOutDatabase = () => {
  state.modalTitle = t("quick-action.transfer-out-db-title");
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.project.database.transfer-out";
  state.showModal = true;
};

const createInstance = () => {
  const instanceList = useInstanceStore().getInstanceList();
  if (subscriptionStore.instanceCount <= instanceList.length) {
    state.featureName = "bb.feature.instance-count";
    state.showFeatureModal = true;
    return;
  }
  state.modalTitle = t("quick-action.create-instance");
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.instance.create";
  state.showModal = true;
};

const alterSchema = () => {
  state.modalTitle = t("database.alter-schema");
  state.quickActionType = "quickaction.bb.database.schema.update";
  state.showModal = true;
};

const changeData = () => {
  state.modalTitle = t("database.change-data");
  state.quickActionType = "quickaction.bb.database.data.update";
  state.showModal = true;
};

const createDatabase = () => {
  state.modalTitle = t("quick-action.create-db");
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.database.create";
  state.showModal = true;
};

const requestDatabase = () => {
  state.modalTitle = "Request database";
  state.modalSubtitle = "";
  state.quickActionType = "quickaction.bb.database.request";
  state.showModal = true;
};

const createEnvironment = () => {
  commandStore.dispatchCommand("bb.environment.create");
};

const createRequestQueryIssue = () => {
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.grant.request",
      role: "QUERIER",
      name: "New grant querier request",
    },
  };
  const routeSlug = routerStore.routeSlug(route);
  const projectSlug = routeSlug.projectSlug;
  if (projectSlug) {
    const id = idFromSlug(projectSlug);
    (routeInfo.query as any).project = id;
  }
  router.push(routeInfo);
};

const createExportDataIssue = () => {
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.grant.request",
      role: "EXPORTER",
      name: "New grant exporter request",
    },
  };
  const routeSlug = routerStore.routeSlug(route);
  const projectSlug = routeSlug.projectSlug;
  if (projectSlug) {
    const id = idFromSlug(projectSlug);
    (routeInfo.query as any).project = id;
  }
  router.push(routeInfo);
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
  "quickaction.bb.database.request": {
    name: t("quick-action.request-db"),
    perform: () => requestDatabase(),
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
