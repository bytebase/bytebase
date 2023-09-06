<template>
  <div class="flex flex-row items-center">
    <template v-if="legacyIssue.status !== 'OPEN' || done">
      <span>-</span>
    </template>
    <template v-else-if="!ready">
      <BBSpin />
    </template>
    <template v-else-if="currentApprover">
      <BBAvatar
        :size="'SMALL'"
        :username="currentApprover.title"
        :email="currentApprover.email"
      />
      <span class="ml-2">
        {{ currentApprover.title }}
      </span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  extractIssueReviewContext,
  useWrappedReviewSteps,
} from "@/plugins/issue/logic";
import { useAuthStore } from "@/store";
import { Issue as LegacyIssue } from "@/types";
import { Issue } from "@/types/proto/v1/issue_service";

const props = defineProps<{
  legacyIssue: LegacyIssue;
}>();

const issue = computed(() => {
  try {
    return Issue.fromJSON(props.legacyIssue.payload.approval);
  } catch {
    return Issue.fromJSON({});
  }
});

const context = extractIssueReviewContext(
  computed(() => props.legacyIssue),
  issue
);
const { ready, done } = context;
const currentUserName = computed(() => useAuthStore().currentUser.name);
const legacyIssue = computed(() => props.legacyIssue);
const wrappedSteps = useWrappedReviewSteps(legacyIssue, context);

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
