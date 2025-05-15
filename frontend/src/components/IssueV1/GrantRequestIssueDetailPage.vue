<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <BannerSection v-if="!isCreating" />
      <FeatureAttention
        v-else-if="existedDeactivatedInstance"
        type="warning"
        feature="bb.feature.custom-approval"
      />
      <HeaderSection />
    </div>

    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden py-2"
      >
        <div class="w-full px-4">
          <GrantRequestExporterForm
            v-if="requestRole === PresetRoleType.PROJECT_EXPORTER"
          />
          <GrantRequestQuerierForm
            v-if="requestRole === PresetRoleType.SQL_EDITOR_USER"
          />
        </div>

        <DescriptionSection />

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
import { computed, ref } from "vue";
import { FeatureAttention } from "@/components/FeatureGuard";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { PresetRoleType } from "@/types";
import { Drawer } from "../v2";
import {
  BannerSection,
  HeaderSection,
  DescriptionSection,
  IssueCommentSection,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  GrantRequestExporterForm,
  GrantRequestQuerierForm,
  Sidebar,
} from "./components";
import { provideIssueIntanceContext } from "./components/Sidebar/ReviewSection/utils";
import type { IssueReviewAction, IssueStatusAction } from "./logic";
import {
  provideIssueSidebarContext,
  useIssueContext,
  usePollIssue,
} from "./logic";

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
} = provideIssueSidebarContext(containerRef);

const { existedDeactivatedInstance } = provideIssueIntanceContext();
</script>
