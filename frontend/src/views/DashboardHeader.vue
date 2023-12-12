<template>
  <div class="flex items-center justify-between h-10 pl-2 pr-4 my-1 space-x-3">
    <div class="flex items-center">
      <BytebaseLogo class="block md:hidden" />

      <div
        class="hidden sm:flex max-w-full w-64 px-2 py-0.5 border border-control-border text-sm rounded-sm items-center justify-between hover:bg-control-bg-hover cursor-pointer ml-2 space-x-2"
        @click="state.showProjectModal = true"
      >
        <ProjectCol
          v-if="project.uid !== `${UNKNOWN_ID}`"
          mode="ALL_SHORT"
          :project="project"
          :show-tenant-icon="true"
        />
        <div v-else class="text-control-placeholder text-sm">
          {{ $t("project.select") }}
        </div>
        <ChevronDownIcon class="h-6 text-gray-400" />
      </div>
    </div>
    <div class="flex-1 flex justify-end items-center space-x-3">
      <button
        class="hidden w-full max-w-xs md:flex items-center justify-between rounded-sm border border-control-border bg-gray-100 hover:bg-control-bg-hover pl-2 pr-1 py-0.5 outline-none"
        @click="onClickSearchButton"
      >
        <span class="text-control-placeholder text-sm">
          {{ $t("common.search") }}
        </span>
        <span class="flex items-center space-x-1">
          <kbd
            class="h-6 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm px-1 text-control overflow-y-hidden"
          >
            <span v-if="isMac" class="text-xl px-0.5">⌘</span>
            <span v-else class="tracking-tighter transform scale-x-90">
              Ctrl
            </span>
            <span class="ml-1 mr-0.5">K</span>
          </kbd>
        </span>
      </button>
      <div
        v-if="currentPlan === PlanType.FREE"
        class="flex justify-between items-center min-w-fit px-4 py-1 bg-emerald-500 text-sm font-medium text-white rounded-md cursor-pointer"
        @click="handleWantHelp"
      >
        <span class="hidden lg:block mr-2">{{ $t("common.want-help") }}</span>
        <heroicons-outline:chat-bubble-left-right class="w-4 h-4" />
      </div>
      <a
        href="/sql-editor"
        target="_blank"
        class="flex items-center text-sm gap-x-1 rounded-sm border border-control-border bg-gray-100 hover:bg-control-bg-hover py-0.5 px-1"
      >
        <heroicons-outline:terminal class="w-6 h-6" />
        <span class="whitespace-nowrap">{{ $t("sql-editor.self") }}</span>
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

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('common.want-help')"
    @close="state.showQRCodeModal = false"
  />

  <ProjectSwitchModal
    v-if="state.showProjectModal"
    :project="project"
    @dismiss="state.showProjectModal = false"
  />
</template>

<script lang="ts" setup>
import { defineAction, useRegisterActions } from "@bytebase/vue-kbar";
import { useKBarHandler } from "@bytebase/vue-kbar";
import { Settings, ChevronDownIcon } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentProject } from "@/components/Project/useCurrentProject";
import {
  useSubscriptionV1Store,
  useInboxV1Store,
  useCurrentUserV1,
} from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { extractUserUID } from "@/utils";
import BytebaseLogo from "../components/BytebaseLogo.vue";
import ProfileBrandingLogo from "../components/ProfileBrandingLogo.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import { useLanguage } from "../composables/useLanguage";
import { UNKNOWN_ID } from "../types";

interface LocalState {
  showQRCodeModal: boolean;
  showProjectModal: boolean;
}

const { t } = useI18n();
const inboxV1Store = useInboxV1Store();
const subscriptionStore = useSubscriptionV1Store();
const router = useRouter();
const { locale } = useLanguage();

const state = reactive<LocalState>({
  showQRCodeModal: false,
  showProjectModal: false,
});

const params = computed(() => {
  const route = router.currentRoute.value;
  return {
    projectSlug: route.params.projectSlug as string,
    issueSlug: route.params.issueSlug as string,
    databaseSlug: route.params.databaseSlug as string,
    changeHistorySlug: route.params.changeHistorySlug as string,
  };
});

const { project } = useCurrentProject(params);

const isMac = navigator.platform.match(/mac/i);
const handler = useKBarHandler();
const onClickSearchButton = () => {
  handler.value.show();
};

const me = useCurrentUserV1();

const { currentPlan } = storeToRefs(subscriptionStore);

const prepareInboxSummary = () => {
  // It will also be called when user logout
  if (extractUserUID(me.value.name) !== String(UNKNOWN_ID)) {
    inboxV1Store.fetchInboxSummary();
  }
};

watchEffect(prepareInboxSummary);

const inboxSummary = computed(() => {
  return inboxV1Store.inboxSummary;
});

const kbarActions = computed(() => {
  return [
    defineAction({
      id: "bb.navigation.global.settings",
      name: "Settings",
      section: t("kbar.navigation"),
      keywords: "navigation",
      perform: () => router.push({ name: "setting.workspace.member" }),
    }),
    defineAction({
      id: "bb.navigation.global.inbox",
      name: "Inbox",
      section: t("kbar.navigation"),
      keywords: "navigation",
      perform: () => router.push({ name: "setting.inbox" }),
    }),
  ];
});
useRegisterActions(kbarActions);

const handleWantHelp = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open("https://www.bytebase.com/docs/faq#how-to-reach-us", "_blank");
  }
};
</script>
