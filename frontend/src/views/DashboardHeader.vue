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
      <NButton class="hidden md:flex" size="small" @click="onClickSearchButton">
        <SearchIcon class="w-4 h-auto mr-1" />
        <span class="text-control-placeholder text-sm mr-2">
          {{ $t("common.search") }}
        </span>
        <span class="flex items-center space-x-1">
          <kbd
            class="h-5 flex items-center justify-center bg-black bg-opacity-10 rounded text-sm px-1 text-control overflow-y-hidden"
          >
            <span v-if="isMac" class="text-lg px-0.5">⌘</span>
            <span v-else class="tracking-tighter transform scale-x-90">
              Ctrl
            </span>
            <span class="ml-1 mr-0.5">K</span>
          </kbd>
        </span>
      </NButton>
      <NButton
        v-if="currentPlan === PlanType.FREE"
        size="small"
        type="success"
        @click="handleWantHelp"
      >
        <span class="hidden lg:block mr-2">{{ $t("common.want-help") }}</span>
        <heroicons-outline:chat-bubble-left-right class="w-4 h-4" />
      </NButton>
      <NButton size="small">
        <a
          href="/sql-editor"
          class="flex flex-row justify-center items-center"
          target="_blank"
        >
          <heroicons-outline:terminal class="w-5 h-auto mr-1" />
          <span class="whitespace-nowrap">{{ $t("sql-editor.self") }}</span>
        </a>
      </NButton>
      <router-link
        v-if="hasGetSettingPermission"
        :to="{ name: SETTING_ROUTE_WORKSPACE_GENERAL }"
        exact-active-class=""
      >
        <SettingsIcon class="w-5 h-auto" />
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
import { SettingsIcon, ChevronDownIcon, SearchIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useCurrentProject } from "@/components/Project/useCurrentProject";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
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
    projectId: route.params.projectId as string,
    issueSlug: route.params.issueSlug as string,
  };
});

const currentUser = useCurrentUserV1();
const { project } = useCurrentProject(params);

const isMac = navigator.platform.match(/mac/i);
const handler = useKBarHandler();
const onClickSearchButton = () => {
  handler.value.show();
};

const { currentPlan } = storeToRefs(subscriptionStore);

const hasGetSettingPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.settings.get");
});

const kbarActions = computed(() => {
  if (!hasGetSettingPermission.value) {
    return [];
  }
  return [
    defineAction({
      id: "bb.navigation.global.settings",
      name: "Settings",
      section: t("kbar.navigation"),
      keywords: "navigation",
      perform: () => router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL }),
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
