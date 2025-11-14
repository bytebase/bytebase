<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <BannerSection v-if="!isCreating" />
      <FeatureAttention :feature="PlanFeature.FEATURE_APPROVAL_WORKFLOW" />
      <HeaderSection />
    </div>

    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden py-2"
      >
        <div
          class="w-full mx-auto flex flex-col justify-start items-start px-4 mb-4 gap-y-4"
        >
          <div
            v-if="requestRole"
            class="w-full flex flex-col justify-start items-start"
          >
            <span class="flex items-center textinfolabel mb-2">
              {{ $t("role.self") }}
            </span>
            <div class="flex flex-row justify-start items-start">
              {{ displayRoleTitle(requestRole) }}
            </div>
          </div>
          <div
            v-if="condition?.databaseResources"
            class="w-full flex flex-col justify-start items-start"
          >
            <span class="flex items-center textinfolabel mb-2">
              {{ $t("common.database") }}
            </span>
            <div
              class="w-full flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
            >
              <span v-if="condition.databaseResources.length === 0">{{
                $t("issue.grant-request.all-databases")
              }}</span>
              <DatabaseResourceTable
                v-else
                class="w-full"
                :database-resource-list="condition.databaseResources"
              />
            </div>
          </div>
          <div class="w-full flex flex-col justify-start items-start">
            <span class="flex items-center textinfolabel mb-2">
              {{ $t("issue.grant-request.expired-at") }}
            </span>
            <div>
              {{
                condition?.expiredTime
                  ? dayjs(new Date(condition.expiredTime)).format("LLL")
                  : $t("project.members.never-expires")
              }}
            </div>
          </div>
        </div>

        <DescriptionSection v-if="isCreating" />
        <IssueCommentSection v-if="!isCreating" />
      </div>

      <div
        v-if="sidebarMode == 'DESKTOP'"
        class="hide-scrollbar border-l"
        :style="{
          width: `${desktopSidebarWidth}px`,
        }"
      >
        <Sidebar />
      </div>
    </div>
  </div>

  <template v-if="sidebarMode === 'MOBILE'">
    <!-- mobile sidebar -->
    <Drawer :show="mobileSidebarOpen" @close="mobileSidebarOpen = false">
      <div
        style="
          min-width: 240px;
          width: 80vw;
          max-width: 320px;
          padding: 0.5rem 0;
        "
      >
        <Sidebar />
      </div>
    </Drawer>
  </template>

  <IssueReviewActionPanel
    :action="ongoingIssueReviewAction?.action"
    @close="ongoingIssueReviewAction = undefined"
  />
  <IssueStatusActionPanel
    :action="ongoingIssueStatusAction?.action"
    @close="ongoingIssueStatusAction = undefined"
  />
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { displayRoleTitle } from "@/utils";
import { convertFromCELString } from "@/utils/issue/cel";
import { provideSidebarContext } from "../Plan/logic";
import { Drawer } from "../v2";
import {
  BannerSection,
  DatabaseResourceTable,
  DescriptionSection,
  HeaderSection,
  IssueCommentSection,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  Sidebar,
} from "./components";
import type { IssueReviewAction, IssueStatusAction } from "./logic";
import { useIssueContext, usePollIssue } from "./logic";

const containerRef = ref<HTMLElement>();
const { isCreating, issue, events } = useIssueContext();

const ongoingIssueReviewAction = ref<{
  action: IssueReviewAction;
}>();
const ongoingIssueStatusAction = ref<{
  action: IssueStatusAction;
}>();

const requestRole = computed(() => {
  return issue.value.grantRequest?.role;
});

const condition = computedAsync(async () => {
  const conditionExpression = await convertFromCELString(
    issue.value.grantRequest?.condition?.expression ?? ""
  );
  return conditionExpression;
});

usePollIssue();

useEmitteryEventListener(
  events,
  "perform-issue-review-action",
  ({ action }) => {
    ongoingIssueReviewAction.value = {
      action,
    };
  }
);

useEmitteryEventListener(
  events,
  "perform-issue-status-action",
  ({ action }) => {
    ongoingIssueStatusAction.value = {
      action,
    };
  }
);

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideSidebarContext(containerRef);
</script>
