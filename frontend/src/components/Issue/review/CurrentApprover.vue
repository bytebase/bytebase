<template>
  <div class="flex flex-row items-center">
    <template v-if="issue.status !== 'OPEN' || done">
      <span>-</span>
    </template>
    <template v-else-if="!ready">
      <BBSpin />
    </template>
    <template v-else-if="currentApprover">
      <BBAvatar :size="'SMALL'" :username="currentApprover.title" />
      <span class="ml-2">
        {{ currentApprover.title }}
      </span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { Issue } from "@/types";
import { Review } from "@/types/proto/v1/issue_service";
import {
  extractIssueReviewContext,
  useWrappedReviewSteps,
} from "@/plugins/issue/logic";
import { useAuthStore } from "@/store";

const props = defineProps<{
  issue: Issue;
}>();

const review = computed(() => {
  try {
    return Review.fromJSON(props.issue.payload.approval);
  } catch {
    return Review.fromJSON({});
  }
});

const context = extractIssueReviewContext(
  computed(() => props.issue),
  review
);
const { ready, done } = context;
const currentUserName = computed(() => useAuthStore().currentUser.name);
const issue = computed(() => props.issue);
const wrappedSteps = useWrappedReviewSteps(issue, context);

const currentStep = computed(() => {
  return wrappedSteps.value?.find((step) => step.status === "CURRENT");
});

const currentApprover = computed(() => {
  if (!currentStep.value) return undefined;
  const me = currentStep.value.candidates.find(
    (user) => user.name === currentUserName.value
  );
  // Show currentUser if currentUser is one of the validate approver candidates.
  if (me) return me;
  // Show the first approver candidate otherwise.
  return currentStep.value.candidates[0];
});
</script>
