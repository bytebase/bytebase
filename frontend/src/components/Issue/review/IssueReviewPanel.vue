<template>
  <div v-if="ready && wrappedSteps" class="mt-1">
    <ApprovalTimeline :steps="wrappedSteps" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import {
  candidatesOfApprovalStep,
  extractUserEmail,
  useAuthStore,
  useUserStore,
} from "@/store";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { useIssueLogic } from "../logic";
import { Issue, WrappedReviewStep } from "@/types";
import ApprovalTimeline from "./ApprovalTimeline.vue";

const userStore = useUserStore();
const issueLogic = useIssueLogic();
const currentUserName = computed(() => useAuthStore().currentUser.name);
const issue = computed(() => issueLogic.issue.value as Issue);
const context = useIssueReviewContext();
const { ready, flow, done } = context;

const wrappedSteps = computed(() => {
  const steps = flow.value.template.flow?.steps;
  const currentStepIndex = flow.value.currentStepIndex ?? -1;

  const statusOfStep = (index: number) => {
    if (done.value) return "DONE";
    if (index < currentStepIndex) return "DONE";
    if (index === currentStepIndex) return "CURRENT";
    return "PENDING";
  };
  const approverOfStep = (index: number) => {
    const principal = flow.value.approvers[index]?.principal;
    if (!principal) return undefined;
    const email = extractUserEmail(principal);
    return userStore.getUserByEmail(email);
  };
  const candidatesOfStep = (index: number) => {
    const step = steps?.[index];
    if (!step) return [];
    const users = candidatesOfApprovalStep(issue.value, step);
    const idx = users.indexOf(currentUserName.value);
    if (idx > 0) {
      users.splice(idx, 1);
      users.unshift(currentUserName.value);
    }
    return users.map((user) => userStore.getUserByName(user)!);
  };

  return steps?.map<WrappedReviewStep>((step, index) => ({
    index,
    step,
    status: statusOfStep(index),
    approver: approverOfStep(index),
    candidates: candidatesOfStep(index),
  }));
});
</script>
