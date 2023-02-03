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
            v-if="showProjectItem"
            to="/setting/project"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("common.projects") }}</router-link
          >
          <router-link
            to="/setting/member"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            :class="[
              route.name === 'workspace.profile' &&
                'router-link-active bg-link-hover',
            ]"
            >{{ $t("settings.sidebar.members") }}</router-link
          >
          <router-link
            v-if="showSensitiveDataItem"
            to="/setting/sensitive-data"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.sensitive-data") }}
          </router-link>
          <router-link
            v-if="showAccessControlItem"
            to="/setting/access-control"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.access-control") }}
          </router-link>
          <router-link
            v-if="showAuditLogItem"
            to="/setting/audit-log"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.audit-log") }}</router-link
          >
          <router-link
            v-if="showIMIntegrationItem"
            to="/setting/im-integration"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.im-integration") }}
            <BBBetaBadge class="ml-1" />
          </router-link>
          <router-link
            v-if="showSSOItem"
            to="/setting/sso"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.sso") }}
            <BBBetaBadge class="ml-1" />
          </router-link>
          <router-link
            v-if="showVCSItem"
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
            v-if="showDebugLogItem"
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

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { hasWorkspacePermission } from "../utils";
import { useCurrentUser, useRouterStore } from "@/store";

const routerStore = useRouterStore();
const route = useRoute();
const router = useRouter();
const currentUser = useCurrentUser();

const showProjectItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-project",
    currentUser.value.role
  );
});

const showSensitiveDataItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sensitive-data",
    currentUser.value.role
  );
});

const showAccessControlItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-access-control",
    currentUser.value.role
  );
});

const showIMIntegrationItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-im-integration",
    currentUser.value.role
  );
});

const showSSOItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sso",
    currentUser.value.role
  );
});

const showVCSItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-vcs-provider",
    currentUser.value.role
  );
});

const showDebugLogItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.debug-log",
    currentUser.value.role
  );
});

const showAuditLogItem = computed((): boolean => {
  return hasWorkspacePermission(
    "bb.permission.workspace.audit-log",
    currentUser.value.role
  );
});

const goBack = () => {
  router.push(routerStore.backPath());
};
</script>
