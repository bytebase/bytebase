<template>
  <PagedIssueTableV1
    method="LIST"
    :session-key="sessionKey"
    :page-size="50"
    :issue-filter="{
      project,
      query: '',
      statusList: [IssueStatus.OPEN],
    }"
  >
    <template #table="{ issueList, loading }">
      <slot
        name="table"
        :issue-list="issueList.filter(filter)"
        :loading="loading"
      />
    </template>
  </PagedIssueTableV1>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useAuthStore } from "@/store";
import type { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import { extractReviewContext, useWrappedReviewStepsV1 } from "../logic";
import PagedIssueTableV1 from "./PagedIssueTableV1.vue";

defineProps<{
  sessionKey: string;
  project: string;
}>();

const currentUserName = computed(() => useAuthStore().currentUser.name);

const filter = (issue: ComposedIssue) => {
  const issueRef = computed(() => issue);
  const reviewContext = extractReviewContext(issueRef);
  const steps = useWrappedReviewStepsV1(issueRef, reviewContext);

  const currentStep = steps.value?.find((step) => step.status === "CURRENT");
  if (!currentStep) return false;
  return (
    currentStep.candidates.findIndex(
      (user) => user.name === currentUserName.value
    ) >= 0
  );
};
</script>
