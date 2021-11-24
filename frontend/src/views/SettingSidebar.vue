<template>
  <!-- Navigation -->
  <nav class="flex-1 flex flex-col px-3 overflow-y-auto">
    <div class="space-y-1">
      <button
        class="
          group
          flex
          items-center
          px-2
          py-2
          text-base
          leading-5
          font-normal
          rounded-md
          text-gray-700
          focus:outline-none
        "
        @click.prevent="goBack"
      >
        <svg
          class="
            mr-1
            h-6
            w-6
            text-gray-500
            group-hover:text-gray-500
            group-focus:text-gray-600
          "
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
          class="
            group
            flex
            items-center
            px-2
            py-2
            text-sm
            leading-5
            font-medium
            rounded-md
            text-gray-700
          "
        >
          <svg
            class="mr-3 w-5 h-5"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              fill-rule="evenodd"
              d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-6-3a2 2 0 11-4 0 2 2 0 014 0zm-2 4a5 5 0 00-4.546 2.916A5.986 5.986 0 0010 16a5.986 5.986 0 004.546-2.084A5 5 0 0010 11z"
              clip-rule="evenodd"
            ></path>
          </svg>
          Account
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/profile"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Profile
          </router-link>
        </div>
      </div>
      <div class="mt-8">
        <div
          class="
            group
            flex
            items-center
            px-2
            py-2
            text-sm
            leading-5
            font-medium
            rounded-md
            text-gray-700
          "
        >
          <svg
            class="mr-3 w-5 h-5"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              fill-rule="evenodd"
              d="M4 4a2 2 0 012-2h8a2 2 0 012 2v12a1 1 0 110 2h-3a1 1 0 01-1-1v-2a1 1 0 00-1-1H9a1 1 0 00-1 1v2a1 1 0 01-1 1H4a1 1 0 110-2V4zm3 1h2v2H7V5zm2 4H7v2h2V9zm2-4h2v2h-2V5zm2 4h-2v2h2V9z"
              clip-rule="evenodd"
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
          <!-- <router-link
            to="/setting/agent"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Agents
          </router-link> -->
          <router-link
            to="/setting/member"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Members
          </router-link>
          <router-link
            v-if="showOwnerItem"
            to="/setting/version-control"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Version Control
          </router-link>
          <router-link
            v-if="false"
            to="/setting/plan"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Plans
          </router-link>
          <!-- <router-link
            to="/setting/billing"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Billing
          </router-link> -->
          <!-- <div class="pl-9 mt-1">
            <BBOutline :title="'Integrations'" :itemList="integrationList" />
          </div> -->
        </div>
      </div>
    </div>
  </nav>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { isOwner } from "../utils";

interface LocalState {
  collapseState: boolean;
}

export default {
  name: "SettingSidebar",
  props: {
    vcsSlug: {
      default: "",
      type: String,
    },
  },
  setup() {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      collapseState: true,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const showOwnerItem = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const goBack = () => {
      router.push(store.getters["router/backPath"]());
    };

    const toggleCollapse = () => {
      state.collapseState = !state.collapseState;
    };

    const integrationList = [
      {
        name: "Slack",
        link: "/setting/integration/slack",
      },
    ];

    return {
      state,
      integrationList,
      showOwnerItem,
      goBack,
      toggleCollapse,
    };
  },
};
</script>
