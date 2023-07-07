<template>
  <div v-if="ready && wrappedSteps" class="mt-1">
    <ApprovalTimeline :steps="wrappedSteps" />
  </div>
</template>

<script lang="ts" setup>
import { useIssueContext, useWrappedReviewStepsV1 } from "@/components/IssueV1";
import ApprovalTimeline from "./ApprovalTimeline.vue";

const { issue, reviewContext } = useIssueContext();
const { ready } = reviewContext;

const wrappedSteps = useWrappedReviewStepsV1(issue, reviewContext);
</script>
