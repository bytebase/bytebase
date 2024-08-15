<template>
  <NTooltip :disabled="errors.length === 0" placement="left">
    <template #trigger>
      {{ option.label }}
    </template>
    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import type { IssueStatusAction } from "@/components/IssueV1/logic";
import {
  allowUserToApplyIssueStatusAction,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { ExtraActionOption } from "../types";

const props = defineProps<{
  option: ExtraActionOption;
}>();

const { issue } = useIssueContext();

const errors = computed(() => {
  const errors: string[] = [];
  const { type, action } = props.option;
  if (type === "ISSUE") {
    const [ok, reason] = allowUserToApplyIssueStatusAction(
      issue.value,
      action as IssueStatusAction
    );
    if (!ok) {
      errors.push(reason);
    }
  }
  return errors;
});
</script>
