<template>
  <div ref="containerRef" class="h-full flex flex-col">
    <div class="border-b">
      <div class="issue-debug">phase: {{ phase }}</div>
      <BannerSection v-if="!isCreating" />
      <HeaderSection />
    </div>

    <div class="flex-1 flex flex-row">
      <div
        class="flex-1 flex flex-col hide-scrollbar divide-y overflow-x-hidden px-4 py-2"
      >
        <GrantRequestExporterForm
          v-if="requestRole === PresetRoleType.EXPORTER"
        />
        <GrantRequestQuerierForm
          v-if="requestRole === PresetRoleType.QUERIER"
        />

        <DescriptionSection />

        <ActivitySection v-if="!isCreating" />
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
import { computed, ref } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { PresetRoleType } from "@/types";
import {
  BannerSection,
  HeaderSection,
  DescriptionSection,
  ActivitySection,
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  GrantRequestExporterForm,
  GrantRequestQuerierForm,
} from "./components";
import {
  IssueReviewAction,
  IssueStatusAction,
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
</script>
