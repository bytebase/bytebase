<template>
  <div v-if="ready && wrappedSteps" class="mt-1">
    <ApprovalTimeline :steps="wrappedSteps" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { useIssueLogic } from "../logic";
import { Issue } from "@/types";
import ApprovalTimeline from "./ApprovalTimeline.vue";
import { useWrappedReviewSteps } from "@/plugins/issue/logic";

const issueLogic = useIssueLogic();
const issue = computed(() => issueLogic.issue.value as Issue);
const context = useIssueReviewContext();
const { ready } = context;

const wrappedSteps = useWrappedReviewSteps(issue, context);
</script>
