<template>
  <PagedIssueTable :page-size="20">
    <template #table="{ issueList, loading }">
      <slot name="table" :issue-list="filter(issueList)" :loading="loading" />
    </template>
  </PagedIssueTable>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Issue } from "@/types";
import { useAuthStore } from "@/store";
import {
  extractIssueReviewContext,
  useWrappedReviewSteps,
} from "@/plugins/issue/logic";
import { Review } from "@/types/proto/v1/issue_service";

const currentUserName = computed(() => useAuthStore().currentUser.name);

const filter = (issueList: Issue[]) => {
  return issueList.filter((issue) => {
    const review = computed(() => {
      try {
        return Review.fromJSON(issue.payload.approval);
      } catch {
        return Review.fromJSON({});
      }
    });
    const context = extractIssueReviewContext(
      computed(() => issue),
      review
    );
    const steps = useWrappedReviewSteps(issue, context);
    const currentStep = steps.value?.find((step) => step.status === "CURRENT");
    if (!currentStep) return false;
    return (
      currentStep.candidates.findIndex(
        (user) => user.name === currentUserName.value
      ) >= 0
    );
  });
};
</script>
