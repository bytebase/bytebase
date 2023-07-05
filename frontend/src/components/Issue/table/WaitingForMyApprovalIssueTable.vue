<template>
  <PagedIssueTable :page-size="20">
    <template #table="{ issueList, loading }">
      <slot name="table" :issue-list="filter(issueList)" :loading="loading" />
    </template>
  </PagedIssueTable>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Issue as LegacyIssue } from "@/types";
import { useAuthStore } from "@/store";
import {
  extractIssueReviewContext,
  useWrappedReviewSteps,
} from "@/plugins/issue/logic";
import { Issue } from "@/types/proto/v1/issue_service";

const currentUserName = computed(() => useAuthStore().currentUser.name);

const filter = (issueList: LegacyIssue[]) => {
  return issueList.filter((legacyIssue) => {
    const issue = computed(() => {
      try {
        return Issue.fromJSON(legacyIssue.payload.approval);
      } catch {
        return Issue.fromJSON({});
      }
    });
    const context = extractIssueReviewContext(
      computed(() => legacyIssue),
      issue
    );
    const steps = useWrappedReviewSteps(legacyIssue, context);
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
