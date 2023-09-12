<template>
  <div>
    <div class="issue-debug">phase: {{ phase }}</div>

    <BannerSection v-if="!isCreating" />

    <HeaderSection class="!border-t-0" />

    <div class="w-full border-t mt-2" />

    <div class="px-4 mt-2">
      <GrantRequestExporterForm
        v-if="requestRole === PresetRoleType.EXPORTER"
      />
      <GrantRequestQuerierForm v-if="requestRole === PresetRoleType.QUERIER" />
    </div>

    <div class="w-full border-t mt-4" />

    <DescriptionSection />

    <div class="w-full border-t mt-4" />

    <ActivitySection v-if="!isCreating" />

    <IssueReviewActionPanel
      :action="ongoingIssueReviewAction?.action"
      @close="ongoingIssueReviewAction = undefined"
    />
    <IssueStatusActionPanel
      :action="ongoingIssueStatusAction?.action"
      @close="ongoingIssueStatusAction = undefined"
    />
  </div>

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
  useIssueContext,
  usePollIssue,
} from "./logic";

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
</script>
