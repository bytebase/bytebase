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
import { NTooltip, NButton } from "naive-ui";
import { computed } from "vue";
import { ErrorList } from "@/components/IssueV1/components/common";
import {
  IssueStatusAction,
  allowUserToApplyIssueStatusAction,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";

const props = defineProps<{
  action: IssueStatusAction;
}>();

defineEmits<{
  (event: "perform-action", action: IssueStatusAction): void;
}>();

const { issue } = useIssueContext();
const currentUser = useCurrentUserV1();

const errors = computed(() => {
  const errors: string[] = [];
  if (
    !allowUserToApplyIssueStatusAction(
      issue.value,
      currentUser.value,
      props.action
    )
  ) {
    errors.push("You are not allowed to perform this action.");
  }
  return errors;
});
</script>
