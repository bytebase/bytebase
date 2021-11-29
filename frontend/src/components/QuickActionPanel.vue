<template>
  <div
    class="
      pt-1
      overflow-hidden
      grid grid-cols-4
      gap-x-2 gap-y-4
      md:inline-flex md:gap-x-0
    "
  >
    <template v-for="(quickAction, index) in quickActionList" :key="index">
      <div
        v-if="quickAction == 'quickaction.bb.instance.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createInstance">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 6v6m0 0v6m0-6h6m-6 0H6"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-base font-normal text-main tracking-tight">
          Add Instance
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.user.manage'"
        class="flex flex-col items-center w-28"
      >
        <router-link to="/setting/member" class="btn-icon-primary p-3">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
            ></path>
          </svg>
        </router-link>
        <h3
          class="
            mt-1
            text-center text-base
            font-normal
            text-main
            tracking-tight
          "
        >
          Manage User
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.query'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            ></path>
          </svg>
        </button>
        <h3
          class="
            mt-1
            text-center text-base
            font-normal
            text-main
            tracking-tight
          "
        >
          Query
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.data.edit'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            ></path>
          </svg>
        </button>
        <h3
          class="
            mt-1
            text-center text-base
            font-normal
            text-main
            tracking-tight
          "
        >
          Edit Data
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createDatabase">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
            ></path>
          </svg>
        </button>
        <h3
          class="
            mt-1
            text-base text-center
            font-normal
            text-main
            tracking-tight
          "
        >
          New DB
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.request'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="requestDatabase">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
            ></path>
          </svg>
        </button>
        <h3
          class="
            mt-1
            text-base text-center
            font-normal
            text-main
            tracking-tight
          "
        >
          Request DB
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.schema.update'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="alterSchema">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Alter Schema
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.database.troubleshoot'"
        class="flex flex-col items-center w-28"
      >
        <router-link to="/issue/new" class="btn-icon-primary p-3">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M7 11.5V14m0-2.5v-6a1.5 1.5 0 113 0m-3 6a1.5 1.5 0 00-3 0v2a7.5 7.5 0 0015 0v-5a1.5 1.5 0 00-3 0m-6-3V11m0-5.5v-1a1.5 1.5 0 013 0v1m0 0V11m0-5.5a1.5 1.5 0 013 0v3m0 0V11"
            ></path>
          </svg>
        </router-link>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Troubleshoot
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.environment.create'"
        class="flex flex-col items-center w-36"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createEnvironment">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 6v6m0 0v6m0-6h6m-6 0H6"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Add Environment
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
          <svg
            class="transform rotate-90 w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 9l4-4 4 4m0 6l-4 4-4-4"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Reorder
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.project.create'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="createProject">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          New Project
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.project.default'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="goDefaultProject">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Default Project
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bb.project.database.transfer'"
        class="flex flex-col items-center w-28"
      >
        <button class="btn-icon-primary p-3" @click.prevent="transferDatabase">
          <svg
            class="w-6 h-6"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 13l-7 7-7-7m14-8l-7 7-7-7"
            ></path>
          </svg>
        </button>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Transfer in DB
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
      <InstanceForm :create="true" @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bb.database.schema.update'
      "
    >
      <!-- eslint-disable vue/attribute-hyphenation -->
      <AlterSchemaPrepForm
        :projectID="projectID"
        @dismiss="state.showModal = false"
      />
    </template>
    <template
      v-else-if="state.quickActionType == 'quickaction.bb.database.create'"
    >
      <!-- eslint-disable vue/attribute-hyphenation -->
      <CreateDatabasePrepForm
        :projectID="projectID"
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
        :projectID="projectID"
        @dismiss="state.showModal = false"
      />
    </template>
  </BBModal>
</template>

<script lang="ts">
import { reactive, PropType, computed } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import ProjectCreate from "../components/ProjectCreate.vue";
import InstanceForm from "../components/InstanceForm.vue";
import AlterSchemaPrepForm from "../components/AlterSchemaPrepForm.vue";
import CreateDatabasePrepForm from "../components/CreateDatabasePrepForm.vue";
import RequestDatabasePrepForm from "../components/RequestDatabasePrepForm.vue";
import TransferDatabaseForm from "../components/TransferDatabaseForm.vue";
import { DEFAULT_PROJECT_ID, ProjectID, QuickActionType } from "../types";
import { idFromSlug } from "../utils";

interface LocalState {
  showModal: boolean;
  modalTitle: string;
  modalSubtitle: string;
  quickActionType: QuickActionType;
}

export default {
  name: "QuickActionPanel",
  components: {
    ProjectCreate,
    InstanceForm,
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
  setup() {
    const router = useRouter();
    const store = useStore();

    const state = reactive<LocalState>({
      showModal: false,
      modalTitle: "",
      modalSubtitle: "",
      quickActionType: "quickaction.bb.instance.create",
    });

    const projectID = computed((): ProjectID | undefined => {
      if (router.currentRoute.value.name == "workspace.project.detail") {
        const parts = router.currentRoute.value.path.split("/");
        return idFromSlug(parts[parts.length - 1]);
      }
      return undefined;
    });

    const createProject = () => {
      state.modalTitle = "Create project";
      state.modalSubtitle = "";
      state.quickActionType = "quickaction.bb.project.create";
      state.showModal = true;
    };

    const goDefaultProject = () => {
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: DEFAULT_PROJECT_ID,
        },
      });
    };

    const transferDatabase = () => {
      state.modalTitle = "Transfer in database from other projects";
      state.modalSubtitle = "";
      state.quickActionType = "quickaction.bb.project.database.transfer";
      state.showModal = true;
    };

    const createInstance = () => {
      state.modalTitle = "Create instance";
      state.modalSubtitle = "";
      state.quickActionType = "quickaction.bb.instance.create";
      state.showModal = true;
    };

    const alterSchema = () => {
      state.modalTitle = "Alter schema";
      state.modalSubtitle = "Choose database";
      state.quickActionType = "quickaction.bb.database.schema.update";
      state.showModal = true;
    };

    const createDatabase = () => {
      state.modalTitle = "Create database";
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
      store.dispatch("command/dispatchCommand", "bb.environment.create");
    };

    const reorderEnvironment = () => {
      store.dispatch("command/dispatchCommand", "bb.environment.reorder");
    };

    return {
      state,
      projectID,
      createProject,
      goDefaultProject,
      transferDatabase,
      createInstance,
      alterSchema,
      createDatabase,
      requestDatabase,
      createEnvironment,
      reorderEnvironment,
    };
  },
};
</script>
