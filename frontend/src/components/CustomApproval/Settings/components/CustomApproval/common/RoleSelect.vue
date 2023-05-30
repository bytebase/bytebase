<template>
  <NSelect
    style="width: 12rem"
    :options="options"
    :value="value.groupValue ?? value.role"
    :placeholder="$t('custom-approval.approval-flow.node.select-approver')"
    :consistent-menu-width="false"
    :disabled="!allowAdmin"
    @update:value="handleUpdate"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { type SelectOption, type SelectGroupOption, NSelect } from "naive-ui";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";

import {
  ApprovalNode,
  ApprovalNode_GroupValue,
  ApprovalNode_Type,
} from "@/types/proto/v1/review_service";
import { useCustomApprovalContext } from "../context";
import { approvalNodeGroupValueText, approvalNodeRoleText } from "@/utils";
import { useRoleStore } from "@/store";
import { isCustomRole } from "@/types";

interface ApprovalNodeSelectOption extends SelectOption {
  value: ApprovalNode_GroupValue | string | undefined;
  node: ApprovalNode;
}

defineProps<{
  value: ApprovalNode;
}>();

const emit = defineEmits<{
  (event: "update:value", value: ApprovalNode): void;
}>();

const { t } = useI18n();
const context = useCustomApprovalContext();
const { allowAdmin } = context;
const { roleList } = storeToRefs(useRoleStore());

const options = computed(() => {
  const presetGroupValueNodes = [
    ApprovalNode_GroupValue.PROJECT_MEMBER,
    ApprovalNode_GroupValue.PROJECT_OWNER,
    ApprovalNode_GroupValue.WORKSPACE_DBA,
    ApprovalNode_GroupValue.WORKSPACE_OWNER,
  ].map<ApprovalNodeSelectOption>((role) => ({
    node: {
      type: ApprovalNode_Type.ANY_IN_GROUP,
      groupValue: role,
    },
    label: approvalNodeGroupValueText(role),
    value: role,
  }));

  const customRoleNodes = roleList.value
    .filter((role) => isCustomRole(role.name))
    .map<ApprovalNodeSelectOption>((role) => ({
      node: {
        type: ApprovalNode_Type.ANY_IN_GROUP,
        role: role.name,
      },
      label: approvalNodeRoleText(role.name),
      value: role.name,
    }));

  if (customRoleNodes.length > 0) {
    const system: SelectGroupOption = {
      type: "group",
      label: t("custom-approval.approval-flow.node.roles.system"),
      key: "system",
      children: presetGroupValueNodes,
    };
    const custom: SelectGroupOption = {
      type: "group",
      label: t("custom-approval.approval-flow.node.roles.custom"),
      key: "custom",
      children: customRoleNodes,
    };
    return [system, custom];
  }

  return presetGroupValueNodes;
});

const handleUpdate = (
  value: ApprovalNodeSelectOption["value"],
  option: SelectOption
) => {
  const { node } = option as ApprovalNodeSelectOption;
  emit("update:value", node);
};
</script>
