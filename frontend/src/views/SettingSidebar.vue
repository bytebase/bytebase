<template>
  <!-- Navigation -->
  <nav class="flex-1 flex flex-col px-3 overflow-y-auto">
    <div class="space-y-1">
      <button
        class="group flex items-center px-2 py-2 text-base leading-5 font-normal rounded-md text-gray-700 focus:outline-none"
        @click.prevent="goBack"
      >
        <heroicons-outline:chevron-left
          class="mr-1 h-6 w-6 text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
        />
        {{ $t("common.back") }}
      </button>

      <div class="mt-8">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <heroicons-solid:user-circle class="mr-3 w-5 h-5" />
          {{ $t("settings.sidebar.account") }}
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/profile"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.profile") }}</router-link
          >
        </div>
      </div>
      <div class="mt-8">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <heroicons-solid:office-building class="mr-3 w-5 h-5" />
          {{ $t("settings.sidebar.workspace") }}
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/general"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.general") }}
          </router-link>
          <!-- <router-link
            to="/setting/agent"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Agents
          </router-link>-->
          <router-link
            v-if="showOwnerOrDBAItem"
            to="/setting/project"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("common.projects") }}</router-link
          >
          <router-link
            to="/setting/member"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.members") }}</router-link
          >
          <!--
            Label Management is visible to all
            but only editable to Owners and DBAs
          -->
          <router-link
            to="/setting/label"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.labels") }}</router-link
          >
          <router-link
            v-if="showOwnerItem"
            to="/setting/version-control"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.version-control") }}</router-link
          >
          <router-link
            to="/setting/sql-review"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("sql-review.title") }}</router-link
          >
          <router-link
            to="/setting/subscription"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.subscription") }}</router-link
          >
          <router-link
            v-if="showOwnerOrDBAItem"
            to="/setting/debug-log"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.debug-log") }}</router-link
          >
          <!-- <router-link
            to="/setting/billing"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
          >
            Billing
          </router-link>-->
          <!-- <div class="pl-9 mt-1">
            <BBOutline :title="'Integrations'" :itemList="integrationList" />
          </div>-->
        </div>
      </div>
    </div>
  </nav>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useRouter } from "vue-router";
import { isOwner, isDBAOrOwner, isDev } from "../utils";
import { useCurrentUser, useRouterStore } from "@/store";

interface LocalState {
  collapseState: boolean;
}

export default defineComponent({
  name: "SettingSidebar",
  props: {
    vcsSlug: {
      default: "",
      type: String,
    },
    sqlReviewPolicySlug: {
      default: "",
      type: String,
    },
  },
  setup() {
    const routerStore = useRouterStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      collapseState: true,
    });

    const currentUser = useCurrentUser();

    const showOwnerItem = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const showOwnerOrDBAItem = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const goBack = () => {
      router.push(routerStore.backPath());
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
      isDev,
      integrationList,
      showOwnerItem,
      showOwnerOrDBAItem,
      goBack,
      toggleCollapse,
    };
  },
});
</script>
