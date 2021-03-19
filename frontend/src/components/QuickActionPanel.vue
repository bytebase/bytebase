<template>
  <div class="pt-1 overflow-hidden grid grid-cols-3 gap-2 sm:inline-flex">
    <div
      v-if="quickActionList.includes('instance.create')"
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
      v-if="quickActionList.includes('user.manage')"
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
      v-if="quickActionList.includes('datasource.query')"
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
      v-if="quickActionList.includes('datasource.data.edit')"
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
      v-if="quickActionList.includes('datasource.request')"
      class="flex flex-col items-center w-28"
    >
      <router-link
        to="/task/new?template=bytebase.datasource.request"
        class="p-3 btn-icon-primary"
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
            d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
          ></path>
        </svg>
      </router-link>
      <h3 class="mt-1 text-base font-normal text-main tracking-tight">
        Request DB
      </h3>
    </div>

    <div
      v-if="quickActionList.includes('datasource.schema.update')"
      class="flex flex-col items-center w-28"
    >
      <router-link
        to="/task/new?template=bytebase.datasource.schema.update"
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
            d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"
          ></path>
        </svg>
      </router-link>
      <h3 class="mt-1 text-center text-base font-normal text-main">
        Edit Schema
      </h3>
    </div>

    <div
      v-if="quickActionList.includes('ticket.create')"
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
            d="M17.657 18.657A8 8 0 016.343 7.343S7 9 9 10c0-2 .5-5 2.986-7C14 5 16.09 5.777 17.656 7.343A7.975 7.975 0 0120 13a7.975 7.975 0 01-2.343 5.657z"
          ></path>
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9.879 16.121A3 3 0 1012.015 11L11 14H9c0 .768.293 1.536.879 2.121z"
          ></path>
        </svg>
      </router-link>
      <h3 class="mt-1 text-center text-base font-normal text-main">
        Troubleshoot
      </h3>
    </div>

    <div
      v-if="quickActionList.includes('environment.create')"
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
      v-if="quickActionList.includes('environment.reorder')"
      class="flex flex-col items-center w-28"
    >
      <button class="btn-icon-primary p-3" @click.prevent="reorderEnvironment">
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
      <h3 class="mt-1 text-center text-base font-normal text-main">Reorder</h3>
    </div>
  </div>
  <BBModal
    v-if="state.showCreateInstanceModal"
    :title="'Create Instance'"
    @close="state.showCreateInstanceModal = false"
  >
    <InstanceForm
      :create="true"
      @dismiss="state.showCreateInstanceModal = false"
    />
  </BBModal>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import InstanceForm from "../components/InstanceForm.vue";

interface LocalState {
  showCreateInstanceModal: Boolean;
}

export default {
  name: "QuickActionPanel",
  components: {
    InstanceForm,
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
      showCreateInstanceModal: false,
    });

    const createInstance = () => {
      state.showCreateInstanceModal = true;
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
      createEnvironment,
      reorderEnvironment,
    };
  },
};
</script>
