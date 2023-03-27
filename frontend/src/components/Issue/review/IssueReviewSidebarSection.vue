<template>
  <template v-if="ready && steps.length > 0">
    <h2 class="textlabel flex items-start col-span-1 col-start-1 pt-1">
      <div class="flex items-center gap-x-1">
        <span>{{ $t("issue.approval-flow.self") }}</span>
        <NTooltip>
          <template #trigger>
            <heroicons-outline:question-mark-circle />
          </template>
          <div>{{ $t("issue.approval-flow.tooltip") }}</div>
        </NTooltip>
      </div>
    </h2>
    <div class="col-span-2">
      <IssueReviewPanel />
    </div>
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import IssueReviewPanel from "./IssueReviewPanel.vue";

const context = useIssueReviewContext();
const { ready, flow } = context;

const steps = computed(() => flow.value.template.flow?.steps ?? []);
</script>
