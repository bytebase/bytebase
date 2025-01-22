<template>
  <div v-if="!isCreating" class="flex flex-col gap-y-2">
    <div class="textlabel flex items-center gap-x-1">
      {{ $t("issue.approval-flow.self") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useIssueContext } from "@/components/IssueV1";

const { issue, isCreating } = useIssueContext();
computed(() => {
  console.log("raw", issue.value.planCheckRunList);
  return issue.value.planCheckRunList;
});
</script>
