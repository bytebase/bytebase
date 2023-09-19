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
import {
  IssueStatusAction,
  allowUserToApplyIssueStatusAction,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";
import { ExtraActionOption } from "../types";

const props = defineProps<{
  option: ExtraActionOption;
}>();

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();

const errors = computed(() => {
  const errors: string[] = [];
  const { type, action } = props.option;
  if (type === "ISSUE") {
    if (
      !allowUserToApplyIssueStatusAction(
        issue.value,
        currentUser.value,
        action as IssueStatusAction
      )
    ) {
      errors.push("You are not allowed to perform this action");
    }
  }
  return errors;
});
</script>
