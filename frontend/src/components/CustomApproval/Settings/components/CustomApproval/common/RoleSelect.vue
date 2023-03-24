<template>
  <NSelect
    style="width: 12rem"
    :options="options"
    :value="value"
    :placeholder="$t('custom-approval.approval-flow.select')"
    :consistent-menu-width="false"
    :disabled="!allowAdmin"
    @update:value="$emit('update:value', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { type SelectOption, type SelectProps, NSelect } from "naive-ui";

import { ApprovalNode_GroupValue } from "@/types/proto/store/approval";
import { useCustomApprovalContext } from "../context";
import { approvalNodeGroupValueText } from "@/utils";

const context = useCustomApprovalContext();
const { allowAdmin } = context;

interface RoleSelectorProps extends SelectProps {
  value?: ApprovalNode_GroupValue;
}
defineProps<RoleSelectorProps>();

defineEmits<{
  (event: "update:value", value: ApprovalNode_GroupValue): void;
}>();

const options = computed(() => {
  return [
    ApprovalNode_GroupValue.PROJECT_MEMBER,
    ApprovalNode_GroupValue.PROJECT_OWNER,
    ApprovalNode_GroupValue.WORKSPACE_DBA,
    ApprovalNode_GroupValue.WORKSPACE_OWNER,
  ].map<SelectOption>((role) => ({
    label: approvalNodeGroupValueText(role),
    value: role,
  }));
});
</script>
