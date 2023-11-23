<template>
  <div class="flex items-center justify-between h-10 pl-2 pr-4 space-x-2">
    <div class="flex-1 flex items-center">
      <BytebaseLogo class="block md:hidden" />

      <div class="flex-1 hidden md:block">
        <button
          class="w-full max-w-md flex items-center justify-between rounded-md border border-control-border bg-white hover:bg-control-bg-hover pl-2 pr-1 py-0.5 outline-none"
          @click="onClickSearchButton"
        >
          <span class="text-control-placeholder">
            {{ $t("common.search") }}
          </span>
          <span class="flex items-center space-x-1">
            <kbd
              class="h-5 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm px-1 text-control overflow-y-hidden"
            >
              <span v-if="isMac" class="text-xl px-0.5">⌘</span>
              <span v-else class="tracking-tighter transform scale-x-90">
                Ctrl
              </span>
              <span class="ml-1 mr-0.5">K</span>
            </kbd>
          </span>
        </button>
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
        <router-link to="/setting/general" exact-active-class="">
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
      </div>
    </div>
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
import { useKBarHandler } from "@bytebase/vue-kbar";
import { Settings } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  useCurrentUser,
  useSubscriptionV1Store,
  useInboxV1Store,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import BytebaseLogo from "../components/BytebaseLogo.vue";
import ProfileBrandingLogo from "../components/ProfileBrandingLogo.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import { useLanguage } from "../composables/useLanguage";
import { UNKNOWN_ID } from "../types";

interface LocalState {
  showQRCodeModal: boolean;
}

const { t } = useI18n();
const inboxV1Store = useInboxV1Store();
const subscriptionStore = useSubscriptionV1Store();
const router = useRouter();
const { locale } = useLanguage();

const state = reactive<LocalState>({
  showQRCodeModal: false,
});

const isMac = navigator.platform.match(/mac/i);
const handler = useKBarHandler();

const onClickSearchButton = () => {
  handler.value.show();
};

const currentUser = useCurrentUser();

const { currentPlan } = storeToRefs(subscriptionStore);

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
