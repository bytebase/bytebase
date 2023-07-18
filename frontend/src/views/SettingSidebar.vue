<template>
  <!-- Navigation -->
  <nav class="flex-1 flex flex-col overflow-y-hidden">
    <BytebaseLogo class="w-full px-4 shrink-0" />
    <div class="space-y-1 flex-1 overflow-y-auto px-2 pb-4">
      <button
        class="group shrink-0 flex items-center px-2 py-2 text-base leading-5 font-normal rounded-md text-gray-700 hover:opacity-80 focus:outline-none"
        @click.prevent="goBack"
      >
        <heroicons-outline:chevron-left
          class="mr-1 w-5 h-auto text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
        />
        {{ $t("common.back") }}
      </button>
      <div>
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
      <div>
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
            to="/setting/role"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.custom-roles") }}</router-link
          >
          <router-link
            v-if="showProjectItem"
            to="/setting/project"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("common.projects") }}</router-link
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
        </div>
      </div>
      <div>
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <heroicons-solid:shield-check class="mr-3 w-5 h-5" />
          {{ $t("settings.sidebar.security-and-policy") }}
        </div>
        <div class="space-y-1">
          <router-link
            to="/setting/sql-review"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("sql-review.title") }}</router-link
          >
          <router-link
            to="/setting/slow-query"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2 capitalize"
            >{{ $t("slow-query.self") }}</router-link
          >
          <router-link
            to="/setting/schema-template"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2 capitalize"
            >{{ $t("schema-template.self") }}</router-link
          >
          <router-link
            to="/setting/risk-center"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("custom-approval.risk.risk-center") }}
          </router-link>
          <router-link
            to="/setting/custom-approval"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("custom-approval.self") }}
          </router-link>
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
        </div>
      </div>
      <div v-if="showIntegrationSection">
        <div
          class="group flex items-center px-2 py-2 text-sm leading-5 font-medium rounded-md text-gray-700"
        >
          <heroicons-solid:link class="mr-3 w-5 h-5" />
          {{ $t("settings.sidebar.integration") }}
        </div>
        <div class="space-y-1">
          <router-link
            v-if="showVCSItem"
            to="/setting/gitops"
            class="outline-item group w-full flex items-center pl-11 pr-2 py-2"
            >{{ $t("settings.sidebar.gitops") }}</router-link
          >
          <router-link
            v-if="showSSOItem"
            to="/setting/sso"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.sso") }}
          </router-link>
          <router-link
            v-if="showIMIntegrationItem"
            to="/setting/im-integration"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.im-integration") }}
          </router-link>
          <router-link
            v-if="showMailDeliveryItem"
            to="/setting/mail-delivery"
            class="outline-item group w-full flex items-center truncate pl-11 pr-2 py-2"
          >
            {{ $t("settings.sidebar.mail-delivery") }}
          </router-link>
        </div>
      </div>
    </div>
  </nav>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { hasWorkspacePermissionV1 } from "../utils";
import { useCurrentUserV1, useRouterStore } from "@/store";
import BytebaseLogo from "@/components/BytebaseLogo.vue";

const routerStore = useRouterStore();
const route = useRoute();
const router = useRouter();
const currentUserV1 = useCurrentUserV1();

const showProjectItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-project",
    currentUserV1.value.userRole
  );
});

const showSensitiveDataItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const showAccessControlItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-access-control",
    currentUserV1.value.userRole
  );
});

const showIMIntegrationItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-im-integration",
    currentUserV1.value.userRole
  );
});

const showSSOItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sso",
    currentUserV1.value.userRole
  );
});

const showVCSItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-vcs-provider",
    currentUserV1.value.userRole
  );
});

const showIntegrationSection = computed(() => {
  return showVCSItem.value || showIMIntegrationItem.value || showSSOItem.value;
});

const showDebugLogItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.debug-log",
    currentUserV1.value.userRole
  );
});

const showAuditLogItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.audit-log",
    currentUserV1.value.userRole
  );
});

const showMailDeliveryItem = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-mail-delivery",
    currentUserV1.value.userRole
  );
});

const goBack = () => {
  router.push(routerStore.backPath());
};
</script>
