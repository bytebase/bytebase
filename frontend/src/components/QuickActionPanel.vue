<template>
  <div
    class="pt-1 overflow-hidden grid grid-cols-4 gap-x-2 gap-y-4 md:inline-flex md:gap-x-0"
  >
    <template v-for="(quickAction, index) in quickActionList" :key="index">
      <div
        v-if="quickAction == 'quickaction.bb.instance.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createInstance">
          <heroicons-outline:plus-sm class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-base font-normal text-main tracking-tight">
          {{ $t("quick-action.add-instance") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.user.manage'"
        class="flex flex-col items-center w-28"
      >
        <router-link to="/setting/member" class="btn-icon-primary p-3">
          <heroicons-outline:users class="w-6 h-6" />
        </router-link>
        <h3
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.manage-user") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.query'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3">
          <heroicons-outline:search class="w-6 h-6" />
        </button>
        <h3
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          {{ $t("common.query") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.data.edit'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3">
          <heroicons-outline:pencil-alt class="w-6 h-6" />
        </button>
        <h3
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.edit-data") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createDatabase">
          <heroicons-outline:database class="w-6 h-6" />
        </button>
        <h3
          class="mt-1 text-base text-center font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.new-db") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.request'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="requestDatabase">
          <heroicons-outline:database class="w-6 h-6" />
        </button>
        <h3
          class="mt-1 text-base text-center font-normal text-main tracking-tight"
        >
          {{ $t("quick-action.request-db") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.schema.update'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="alterSchema">
          <heroicons-outline:pencil-alt class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("database.alter-schema") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.data.update'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="changeData">
          <heroicons-outline:pencil class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("database.change-data") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.troubleshoot'"
        class="flex flex-col items-center w-28"
      >
        <router-link to="/issue/new" class="btn-icon-primary p-3">
          <heroicons-outline:hand class="w-6 h-6" />
        </router-link>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("quick-action.troubleshoot") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.environment.create'"
        class="flex flex-col items-center w-36"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createEnvironment">
          <heroicons-outline:plus-sm class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("environment.create") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.environment.reorder'"
        class="flex flex-col items-center w-28"
      >
        <button
          class="btn-icon-primary p-3"
          @click.prevent="reorderEnvironment"
        >
          <heroicons-outline:selector class="transform rotate-90 w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("common.reorder") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.project.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createProject">
          <heroicons-outline:template class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("quick-action.new-project") }}
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.project.database.transfer'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="transferDatabase">
          <heroicons-outline:chevron-double-down class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          {{ $t("quick-action.transfer-in-db") }}
        </h3>
      </div>
    </template>
  </div>
  <BBModal
    v-if="state.showModal"
    :title="state.modalTitle"
    :subtitle="state.modalSubtitle"
    @close="state.showModal = false"
  >
    <template v-if="state.quickActionType == 'quickaction.bb.project.create'">
      <ProjectCreate @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.instance.create'"
    >
      <CreateInstanceForm @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bb.database.schema.update'
      "
    >
      <!-- eslint-disable vue/attribute-hyphenation -->
      <AlterSchemaPrepForm
        :projectId="projectId"
        :type="'bb.issue.database.schema.update'"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.data.update'"
    >
      <!-- eslint-disable vue/attribute-hyphenation -->
      <AlterSchemaPrepForm
        :projectId="projectId"
        :type="'bb.issue.database.data.update'"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.create'"
    >
      <!-- eslint-disable vue/attribute-hyphenation -->
      <CreateDatabasePrepForm
        :projectId="projectId"
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
      <!-- eslint-disable vue/attribute-hyphenation -->
      <TransferDatabaseForm
        :projectId="projectId"
        @dismiss="state.showModal = false"
      />
    </template>
  </BBModal>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.instance-count"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts">
import { defineComponent, reactive, PropType, computed } from "vue";
import { useRouter } from "vue-router";
import ProjectCreate from "../components/ProjectCreate.vue";
import CreateInstanceForm from "../components/CreateInstanceForm.vue";
import AlterSchemaPrepForm from "./AlterSchemaPrepForm/";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import RequestDatabasePrepForm from "../components/RequestDatabasePrepForm.vue";
import TransferDatabaseForm from "../components/TransferDatabaseForm.vue";
import { ProjectId, QuickActionType } from "../types";
import { idFromSlug } from "../utils";
import { Action, defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useI18n } from "vue-i18n";
import {
  useCommandStore,
  useInstanceStore,
  useSubscriptionStore,
} from "@/store";

interface LocalState {
  showModal: boolean;
  showFeatureModal: boolean;
  modalTitle: string;
  modalSubtitle: string;
  quickActionType: QuickActionType;
}

export default defineComponent({
  name: "QuickActionPanel",
  components: {
    ProjectCreate,
    CreateInstanceForm,
    AlterSchemaPrepForm,
    CreateDatabasePrepForm,
    RequestDatabasePrepForm,
    TransferDatabaseForm,
  },
  props: {
    quickActionList: {
      required: true,
      type: Object as PropType<QuickActionType[]>,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const router = useRouter();
    const commandStore = useCommandStore();
    const subscriptionStore = useSubscriptionStore();

    const state = reactive<LocalState>({
      showModal: false,
      showFeatureModal: false,
      modalTitle: "",
      modalSubtitle: "",
      quickActionType: "quickaction.bb.instance.create",
    });

    const projectId = computed((): ProjectId | undefined => {
      if (router.currentRoute.value.name == "workspace.project.detail") {
        const parts = router.currentRoute.value.path.split("/");
        return idFromSlug(parts[parts.length - 1]);
      }
      return undefined;
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

    const createInstance = () => {
      const { subscription } = subscriptionStore;
      const instanceList = useInstanceStore().getInstanceList();
      if ((subscription?.instanceCount ?? 5) <= instanceList.length) {
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
      state.modalSubtitle = t("quick-action.choose-db");
      state.quickActionType = "quickaction.bb.database.schema.update";
      state.showModal = true;
    };

    const changeData = () => {
      state.modalTitle = t("database.change-data");
      state.modalSubtitle = t("quick-action.choose-db");
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

    return {
      state,
      projectId,
      createProject,
      transferDatabase,
      createInstance,
      alterSchema,
      changeData,
      createDatabase,
      requestDatabase,
      createEnvironment,
      reorderEnvironment,
    };
  },
});
</script>
