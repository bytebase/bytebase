<template>
  <div class="flex gap-x-20">
    <SQLRuleTable
      key="sql-review-preview-rule-table"
      class="w-full"
      :editable="false"
      :rule-list="selectedRuleList"
    />
  </div>
</template>

<script lang="ts" setup>
import { RuleTemplate, Environment } from "@/types";
import { SQLRuleTable } from "./components";

withDefaults(
  defineProps<{
    name?: string;
    isPreviewStep?: boolean;
    selectedEnvironment?: Environment;
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    name: "",
    isPreviewStep: false,
    selectedEnvironment: undefined,
    selectedRuleList: () => [],
  }
);
</script>
