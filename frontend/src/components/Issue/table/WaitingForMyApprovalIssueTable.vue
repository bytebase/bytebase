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
import { Review } from "@/types/proto/v1/review_service";

const currentUserName = computed(() => useAuthStore().currentUser.name);

const filter = (issueList: Issue[]) => {
  const issueListWithReview = issueList.map((issue) => {
    const review = computed(() => {
      try {
        return Review.fromJSON(issue.payload.approval);
      } catch {
        return Review.fromJSON({});
      }
    });
    const context = extractIssueReviewContext(review);
    const steps = useWrappedReviewSteps(issue, context);
    return {
      issue,
      context,
      steps,
    };
  });

  const filteredList = issueListWithReview.filter(({ steps }) => {
    const currentStep = steps.value?.find((step) => step.status === "CURRENT");

    const me = currentStep?.candidates.find(
      (user) => user.name === currentUserName.value
    );

    return me;
  });
  return filteredList.map(({ issue }) => issue);
};
</script>
