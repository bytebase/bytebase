<template>
  <div class="flex items-center justify-between h-10 pl-2 pr-4">
    <div class="flex items-center">
      <BytebaseLogo class="block md:hidden" />

      <div class="hidden md:block">
        <div class="flex items-baseline space-x-1 whitespace-nowrap">
          <router-link
            to="/issue"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/issue')"
            >{{ $t("common.issues") }}</router-link
          >

          <router-link
            to="/branches"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/branches')"
            >{{ $t("common.branches") }}</router-link
          >

          <router-link
            to="/project"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/project')"
            data-label="bb-header-project-button"
          >
            {{ $t("common.projects") }}
          </router-link>

          <router-link
            to="/db"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/db')"
            data-label="bb-dashboard-header-database-entry"
            >{{ $t("common.databases") }}</router-link
          >

          <router-link
            v-if="shouldShowInstanceEntry"
            to="/instance"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/instance')"
            >{{ $t("common.instances") }}</router-link
          >

          <router-link
            to="/environment"
            class="bar-link px-2 py-1 rounded-md"
            :class="getRouteLinkClass('/environment')"
            >{{ $t("common.environments") }}</router-link
          >
        </div>
      </div>
    </div>
    <div>
      <div class="flex items-center space-x-3">
        <div
          v-if="currentPlan === PlanType.FREE"
          class="flex justify-between items-center min-w-fit px-4 py-1 bg-emerald-500 text-sm font-medium text-white rounded-md cursor-pointer"
          @click="handleWantHelp"
        >
          <span class="hidden lg:block mr-2">{{ $t("common.want-help") }}</span>
          <heroicons-outline:chat-bubble-left-right class="w-4 h-4" />
        </div>
        <a href="/sql-editor" target="_blank">
          <heroicons-outline:terminal class="w-6 h-6" />
        </a>
        <router-link to="/setting/member" exact-active-class="">
          <Settings class="w-6 h-6" />
        </router-link>
        <router-link to="/inbox" exact-active-class="">
          <span
            v-if="inboxSummary.unread > 0"
            class="absolute rounded-full ml-4 -mt-1 h-2 w-2 bg-accent opacity-75"
          ></span>
          <heroicons-outline:bell class="w-6 h-6" />
        </router-link>
        <div class="ml-2">
          <ProfileBrandingLogo>
            <ProfileDropdown />
          </ProfileBrandingLogo>
        </div>
        <div class="ml-2 -mr-2 flex md:hidden">
          <!-- Mobile menu button -->
          <button
            class="icon-link inline-flex items-center justify-center rounded-md"
            @click.prevent="state.showMobileMenu = !state.showMobileMenu"
          >
            <span class="sr-only">Open main menu</span>
            <!--
              Heroicon name: menu

              Menu open: "hidden", Menu closed: "block"
            -->
            <heroicons-solid:dots-vertical class="w-6 h-6" />
          </button>
        </div>
      </div>
    </div>
  </div>

  <!--
      Mobile menu, toggle classes based on menu state.

      Open: "block", closed: "hidden"
  -->
  <div v-if="state.showMobileMenu" class="block md:hidden">
    <router-link to="/issue" class="bar-link rounded-md block px-3 py-1">
      {{ $t("common.issues") }}
    </router-link>

    <router-link to="/project" class="bar-link rounded-md block px-3 py-1">
      {{ $t("common.projects") }}
    </router-link>

    <router-link to="/db" class="bar-link rounded-md block px-3 py-1">
      {{ $t("common.databases") }}
    </router-link>

    <router-link to="/instance" class="bar-link rounded-md block px-3 py-1">{{
      $t("common.instances")
    }}</router-link>

    <router-link
      to="/environment"
      class="bar-link rounded-md block px-3 py-1"
      >{{ $t("common.environments") }}</router-link
    >

    <router-link
      to="/setting/member"
      class="bar-link rounded-md block px-3 py-1"
      >{{ $t("common.settings") }}</router-link
    >
  </div>
  <BBModal
    v-if="state.showQRCodeModal"
    :title="$t('common.want-help')"
    @close="state.showQRCodeModal = false"
  >
    <img
      class="w-48 h-48"
      src="@/assets/bb-helper-wechat-qrcode.webp"
      alt="bb_helper"
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { Settings } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  useCurrentUser,
  useSubscriptionV1Store,
  useInboxV1Store,
  useCurrentUserV1,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import BytebaseLogo from "../components/BytebaseLogo.vue";
import ProfileBrandingLogo from "../components/ProfileBrandingLogo.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import { useLanguage } from "../composables/useLanguage";
import { UNKNOWN_ID } from "../types";
import { hasWorkspacePermissionV1 } from "../utils";

interface LocalState {
  showMobileMenu: boolean;
  showQRCodeModal: boolean;
}

const { t } = useI18n();
const inboxV1Store = useInboxV1Store();
const subscriptionStore = useSubscriptionV1Store();
const router = useRouter();
const route = useRoute();
const { locale } = useLanguage();

const state = reactive<LocalState>({
  showMobileMenu: false,
  showQRCodeModal: false,
});

const currentUser = useCurrentUser();
const currentUserV1 = useCurrentUserV1();

const { currentPlan } = storeToRefs(subscriptionStore);

const getRouteLinkClass = (prefix: string): string[] => {
  const { path } = route;
  const isActiveRoute = path === prefix || path.startsWith(`${prefix}/`);
  const classes: string[] = [];
  if (isActiveRoute) {
    classes.push("router-link-active", "bg-link-hover");
  }
  return classes;
};

const shouldShowInstanceEntry = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-instance",
    currentUserV1.value.userRole
  );
});

const prepareInboxSummary = () => {
  // It will also be called when user logout
  if (currentUser.value.id != UNKNOWN_ID) {
    inboxV1Store.fetchInboxSummary();
  }
};

watchEffect(prepareInboxSummary);

const inboxSummary = computed(() => {
  return inboxV1Store.inboxSummary;
});

const kbarActions = computed(() => [
  defineAction({
    id: "bb.navigation.projects",
    name: "Projects",
    shortcut: ["g", "p"],
    section: t("kbar.navigation"),
    keywords: "navigation",
    perform: () => router.push({ name: "workspace.project" }),
  }),
  defineAction({
    id: "bb.navigation.databases",
    name: "Databases",
    shortcut: ["g", "d"],
    section: t("kbar.navigation"),
    keywords: "navigation db",
    perform: () => router.push({ name: "workspace.database" }),
  }),
  defineAction({
    id: "bb.navigation.instances",
    name: "Instances",
    shortcut: ["g", "i"],
    section: t("kbar.navigation"),
    keywords: "navigation",
    perform: () => router.push({ name: "workspace.instance" }),
  }),
  defineAction({
    id: "bb.navigation.environments",
    name: "Environments",
    shortcut: ["g", "e"],
    section: t("kbar.navigation"),
    keywords: "navigation",
    perform: () => router.push({ name: "workspace.environment" }),
  }),
  defineAction({
    id: "bb.navigation.settings",
    name: "Settings",
    shortcut: ["g", "s"],
    section: t("kbar.navigation"),
    keywords: "navigation",
    perform: () => router.push({ name: "setting.workspace.member" }),
  }),
  defineAction({
    id: "bb.navigation.inbox",
    name: "Inbox",
    shortcut: ["g", "m"],
    section: t("kbar.navigation"),
    keywords: "navigation",
    perform: () => router.push({ name: "setting.inbox" }),
  }),
]);
useRegisterActions(kbarActions);

const handleWantHelp = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open("https://www.bytebase.com/docs/faq#how-to-reach-us", "_blank");
  }
};
</script>
