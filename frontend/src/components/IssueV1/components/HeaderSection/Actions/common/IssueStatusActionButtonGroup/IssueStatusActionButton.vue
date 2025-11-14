<template>
  <NTooltip :disabled="errors.length === 0" placement="top">
    <template #trigger>
      <NButton
        :disabled="errors.length > 0"
        size="medium"
        tag="div"
        v-bind="issueStatusActionButtonProps(action)"
        @click.prevent="$emit('perform-action', action)"
      >
        {{ issueStatusActionDisplayName(action) }}
      </NButton>
    </template>

    <template #default>
      <ErrorList :errors="errors" />
    </template>
  </NTooltip>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import { computed } from "vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import type { IssueStatusAction } from "@/components/IssueV1/logic";
import {
  allowUserToApplyIssueStatusAction,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";

const props = defineProps<{
  action: IssueStatusAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueStatusAction): void;
}>();

const { issue } = useIssueContext();

const errors = computed(() => {
  const errors: string[] = [];
  const [ok, reason] = allowUserToApplyIssueStatusAction(
    issue.value,
    props.action
  );
  if (!ok) {
    errors.push(reason);
  }
  return errors;
});
</script>
