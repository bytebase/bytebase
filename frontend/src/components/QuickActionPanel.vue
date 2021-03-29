<template>
  <div class="pt-1 overflow-hidden grid grid-cols-3 sm:inline-flex">
    <template v-for="(quickAction, index) in quickActionList" :key="index">
      <div
        v-if="quickAction == 'quickaction.bytebase.instance.create'"
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
        v-if="quickAction == 'quickaction.bytebase.user.manage'"
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
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          Manage User
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bytebase.database.query'"
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
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          Query
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bytebase.database.data.edit'"
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
          class="mt-1 text-center text-base font-normal text-main tracking-tight"
        >
          Edit Data
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bytebase.database.request'"
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
          class="mt-1 text-base text-center font-normal text-main tracking-tight"
        >
          Request DB
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bytebase.database.schema.update'"
        class="flex flex-col items-center w-28"
      >
        <router-link
          to="/task/new?template=bytebase.database.schema.update"
          class="btn-icon-primary p-3"
        >
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
        </router-link>
        <h3 class="mt-1 text-center text-base font-normal text-main">
          Alter Schema
        </h3>
      </div>

      <div
        v-if="quickAction == 'quickaction.bytebase.database.troubleshoot'"
        class="flex flex-col items-center w-28"
      >
        <router-link to="/task/new" class="btn-icon-primary p-3">
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
        v-if="quickAction == 'quickaction.bytebase.environment.create'"
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
        v-if="quickAction == 'quickaction.bytebase.environment.reorder'"
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
    </template>
  </div>
  <BBModal
    v-if="state.showModal"
    :title="state.modalTitle"
    @close="state.showModal = false"
  >
    <template
      v-if="state.quickActionType == 'quickaction.bytebase.instance.create'"
    >
      <InstanceForm :create="true" @dismiss="state.showModal = false" />
    </template>
    <template
      v-else-if="
        state.quickActionType == 'quickaction.bytebase.database.request'
      "
    >
      <RequestDatabasePrepForm @dismiss="state.showModal = false" />
    </template>
  </BBModal>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import InstanceForm from "../components/InstanceForm.vue";
import RequestDatabasePrepForm from "../components/RequestDatabasePrepForm.vue";
import { QuickActionType } from "../types";

interface LocalState {
  showModal: Boolean;
  modalTitle: string;
  quickActionType: QuickActionType;
}

export default {
  name: "QuickActionPanel",
  components: {
    InstanceForm,
    RequestDatabasePrepForm,
  },
  props: {
    quickActionList: {
      required: true,
      type: Object as PropType<String[]>,
    },
  },
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
      showModal: false,
      modalTitle: "",
      quickActionType: "quickaction.bytebase.instance.create",
    });

    const createInstance = () => {
      state.modalTitle = "Create instance";
      state.quickActionType = "quickaction.bytebase.instance.create";
      state.showModal = true;
    };

    const requestDatabase = () => {
      state.modalTitle = "Request database";
      state.quickActionType = "quickaction.bytebase.database.request";
      state.showModal = true;
    };

    const createEnvironment = () => {
      store.dispatch("command/dispatchCommand", "bytebase.environment.create");
    };

    const reorderEnvironment = () => {
      store.dispatch("command/dispatchCommand", "bytebase.environment.reorder");
    };

    return {
      state,
      createInstance,
      requestDatabase,
      createEnvironment,
      reorderEnvironment,
    };
  },
};
</script>
