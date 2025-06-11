<template>
  <div class="flex items-center justify-between h-10 px-4 my-1 space-x-3">
    <BytebaseLogo
      v-if="showLogo"
      class="shrink-0"
      :redirect="WORKSPACE_ROUTE_LANDING"
    />
    <ProjectSwitchPopover />

    <div class="flex-1 flex justify-end items-center space-x-3">
      <NButton
        v-if="currentPlan === PlanType.FREE"
        size="small"
        type="success"
        @click="handleWantHelp"
      >
        <template #icon>
          <MessagesSquareIcon class="w-4 h-4" />
        </template>
        <span class="hidden lg:block">{{ $t("common.want-help") }}</span>
      </NButton>

      <router-link
        :to="sqlEditorLink"
        class="flex flex-row justify-center items-center"
        exact-active-class=""
        target="_blank"
      >
        <NButton size="small">
          <template #icon>
            <SquareTerminalIcon class="w-4 h-auto" />
          </template>
          <span class="whitespace-nowrap">{{ $t("sql-editor.self") }}</span>
        </NButton>
      </router-link>

      <NTooltip :disabled="windowWidth >= 640">
        <template #trigger>
          <router-link :to="myIssueLink" class="flex">
            <NButton size="small" @click="goToMyIssues">
              <template #icon>
                <CircleDotIcon class="w-4" />
              </template>
              <span class="hidden sm:block">{{ $t("issue.my-issues") }}</span>
            </NButton>
          </router-link>
        </template>
        {{ $t("issue.my-issues") }}
      </NTooltip>

      <ProfileBrandingLogo>
        <ProfileDropdown :link="true" />
      </ProfileBrandingLogo>
    </div>
  </div>

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('common.want-help')"
    @close="state.showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { useLocalStorage, useWindowSize } from "@vueuse/core";
import {
  CircleDotIcon,
  SquareTerminalIcon,
  MessagesSquareIcon,
} from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import ProjectSwitchPopover from "@/components/Project/ProjectSwitch/ProjectSwitchPopover.vue";
import { useCurrentProject } from "@/components/Project/useCurrentProject";
import WeChatQRModal from "@/components/WeChatQRModal.vue";
import {
  WORKSPACE_ROUTE_MY_ISSUES,
  WORKSPACE_ROUTE_LANDING,
} from "@/router/dashboard/workspaceRoutes";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/router/sqlEditor";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
} from "@/utils";
import { getComponentIdLocalStorageKey } from "@/utils/localStorage";
import BytebaseLogo from "../components/BytebaseLogo.vue";
import ProfileBrandingLogo from "../components/ProfileBrandingLogo.vue";
import ProfileDropdown from "../components/ProfileDropdown.vue";
import { useLanguage } from "../composables/useLanguage";
import { isValidDatabaseName, isValidProjectName } from "../types";

defineProps<{
  showLogo: boolean;
}>();

interface LocalState {
  showQRCodeModal: boolean;
  showProjectModal: boolean;
}

const subscriptionStore = useSubscriptionV1Store();
const router = useRouter();
const { locale } = useLanguage();
const { record } = useRecentVisit();
const { width: windowWidth } = useWindowSize();

const state = reactive<LocalState>({
  showQRCodeModal: false,
  showProjectModal: false,
});

const params = computed(() => {
  const route = router.currentRoute.value;
  return {
    projectId: route.params.projectId as string | undefined,
    issueSlug: route.params.issueSlug as string | undefined,
    instanceId: route.params.instanceId as string | undefined,
    databaseName: route.params.databaseName as string | undefined,
    changelogId: route.params.changelogId as string | undefined,
  };
});

const { project, database } = useCurrentProject(params);

const { currentPlan } = storeToRefs(subscriptionStore);

const sqlEditorLink = computed(() => {
  if (isValidProjectName(project.value.name)) {
    if (isValidDatabaseName(database.value.name)) {
      const { instanceName, databaseName } = extractDatabaseResourceName(
        database.value.name
      );
      return router.resolve({
        name: SQL_EDITOR_DATABASE_MODULE,
        params: {
          project: extractProjectResourceName(project.value.name),
          instance: instanceName,
          database: databaseName,
        },
      });
    }
    return router.resolve({
      name: SQL_EDITOR_PROJECT_MODULE,
      params: {
        project: extractProjectResourceName(project.value.name),
      },
    });
  }
  return router.resolve({
    name: SQL_EDITOR_HOME_MODULE,
  });
});

const myIssueLink = computed(() => {
  return router.resolve({
    name: WORKSPACE_ROUTE_MY_ISSUES,
  });
});

const goToMyIssues = () => {
  record(myIssueLink.value.fullPath);
  // Trigger page reload manually.
  useLocalStorage<string>(
    getComponentIdLocalStorageKey(WORKSPACE_ROUTE_MY_ISSUES),
    ""
  ).value = uuidv4();
};

const handleWantHelp = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open("https://docs.bytebase.com/faq#how-to-reach-us", "_blank");
  }
};
</script>
