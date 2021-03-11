<template>
  <div class="overflow-hidden grid grid-cols-3 gap-2 sm:inline-flex">
    <div
      v-if="quickActionList.includes('instance.create')"
      class="flex flex-col items-center w-28"
    >
      <router-link to="/instance/new" class="btn-icon">
        <span class="btn-icon-primary inline-flex p-3">
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
        </span>
      </router-link>
      <h3 class="mt-1 text-base font-normal text-main tracking-tight">
        Add Instance
      </h3>
    </div>

    <div
      v-if="quickActionList.includes('user.manage')"
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
            d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
          ></path>
        </svg>
      </button>
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
      v-if="quickActionList.includes('datasource.create')"
      class="flex flex-col items-center w-28"
    >
      <router-link
        to="/task/new?template=bytebase.datasource.create"
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
            d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"
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
            d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"
          ></path>
        </svg>
      </router-link>
      <h3 class="mt-1 text-center text-base font-normal text-main">Ticket</h3>
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
</template>

<script lang="ts">
import { PropType } from "vue";
import { useStore } from "vuex";

export default {
  name: "QuickActionPanel",
  props: {
    quickActionList: {
      required: true,
      type: Object as PropType<String[]>,
    },
  },
  setup() {
    const store = useStore();

    const createEnvironment = () => {
      store.dispatch("command/dispatchCommand", "bytebase.environment.create");
    };

    const reorderEnvironment = () => {
      store.dispatch("command/dispatchCommand", "bytebase.environment.reorder");
    };

    return {
      createEnvironment,
      reorderEnvironment,
    };
  },
};
</script>
