<template>
  <div v-if="ready && wrappedSteps" class="mt-1">
    <ApprovalTimeline :steps="wrappedSteps" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useWrappedReviewSteps } from "@/plugins/issue/logic";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { Issue } from "@/types";
import { useIssueLogic } from "../logic";
import ApprovalTimeline from "./ApprovalTimeline.vue";

const issueLogic = useIssueLogic();
const issue = computed(() => issueLogic.issue.value as Issue);
const context = useIssueReviewContext();
const { ready } = context;

const wrappedSteps = useWrappedReviewSteps(issue, context);
</script>
