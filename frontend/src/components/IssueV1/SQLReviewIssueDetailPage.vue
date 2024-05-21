<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <div class="issue-debug">phase: {{ phase }}</div>
      <BannerSection v-if="!isCreating" />

      <HeaderSection />
    </div>
    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden"
      >
        <ReviewIssueSpecSection />
        <SQLCheckSection v-if="isCreating" />
        <PlanCheckSection v-if="!isCreating" />
        <StatementSection />
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
        <Sidebar v-if="sidebarMode === 'MOBILE'" />
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

  <div class="issue-debug">
    <pre class="text-xs">{{ JSON.stringify(issue, null, "  ") }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { provideSQLCheckContext } from "../SQLCheck";
import { Drawer } from "../v2";
import {
  BannerSection,
  HeaderSection,
  PlanCheckSection,
  StatementSection,
  DescriptionSection,
  IssueCommentSection,
  Sidebar,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  SQLCheckSection,
  ReviewIssueSpecSection,
} from "./components";
import type { IssueReviewAction, IssueStatusAction } from "./logic";
import {
  provideIssueSidebarContext,
  useIssueContext,
  usePollIssue,
} from "./logic";

const containerRef = ref<HTMLElement>();
const { isCreating, phase, issue, events } = useIssueContext();

const ongoingIssueReviewAction = ref<{
  action: IssueReviewAction;
}>();
const ongoingIssueStatusAction = ref<{
  action: IssueStatusAction;
}>();

usePollIssue();

events.on("perform-issue-review-action", ({ action }) => {
  ongoingIssueReviewAction.value = {
    action,
  };
});

events.on("perform-issue-status-action", ({ action }) => {
  ongoingIssueStatusAction.value = {
    action,
  };
});

provideSQLCheckContext();

const {
  mode: sidebarMode,
  desktopSidebarWidth,
  mobileSidebarOpen,
} = provideIssueSidebarContext(containerRef);
</script>
