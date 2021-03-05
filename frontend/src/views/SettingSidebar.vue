<template>
  <!-- Navigation -->
  <nav class="flex-1 flex flex-col px-3 overflow-y-auto">
    <div class="space-y-1">
      <button
        @click.prevent="goBack"
        class="group flex items-center px-2 py-2 text-base leading-5 font-normal rounded-md text-gray-700 focus:outline-none"
      >
        <svg
          class="mr-1 h-6 w-6 text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M15 19l-7-7 7-7"
          ></path>
        </svg>
        Back
      </button>

      <div class="mt-8">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <svg
            class="mr-3 h-6 w-6 text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5.121 17.804A13.937 13.937 0 0112 16c2.5 0 4.847.655 6.879 1.804M15 10a3 3 0 11-6 0 3 3 0 016 0zm6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            ></path>
          </svg>
          Account
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/account/profile"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Profile
          </router-link>
        </div>
      </div>
      <div class="mt-8">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <svg
            class="mr-3 h-6 w-6 text-gray-500"
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
          Workspace
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/general"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            General
          </router-link>
          <router-link
            to="/setting/agent"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Agents
          </router-link>
          <router-link
            to="/setting/member"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Members
          </router-link>
          <router-link
            to="/setting/plan"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Plans
          </router-link>
          <router-link
            to="/setting/billing"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Billing
          </router-link>
          <div class="pl-9 mt-1">
            <BBOutline
              :title="'Integrations'"
              :itemList="integrationList.map((item) => item.name)"
              @click-index="goToIntegration"
            />
          </div>
        </div>
      </div>
      <div class="mt-8">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <svg
            class="mr-3 h-6 w-6 text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            ></path>
          </svg>
          Projects
        </div>
        <!-- <SettingProjectListSidePanel /> -->
      </div>
    </div>
  </nav>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
// import SettingProjectListSidePanel from "../components/SettingProjectListSidePanel.vue";

interface LocalState {
  expandState: boolean;
}

export default {
  name: "SettingSidebar",
  props: {},
  components: {
    // SettingProjectListSidePanel,
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      expandState: true,
    });

    const goBack = () => {
      router.push(store.getters["router/backPath"]());
    };

    const toggleExpand = () => {
      state.expandState = !state.expandState;
    };

    const integrationList = [
      {
        name: "Slack",
        path: "/setting/integration/slack",
      },
    ];

    const goToIntegration = (index: number) => {
      router.push(integrationList[index].path);
    };

    return {
      state,
      integrationList,
      goToIntegration,
      goBack,
      toggleExpand,
    };
  },
};
</script>
