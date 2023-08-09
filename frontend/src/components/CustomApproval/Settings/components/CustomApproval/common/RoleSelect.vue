<template>
  <NSelect
    style="width: 12rem"
    :options="options"
    :value="value.groupValue ?? value.role ?? value.externalNodeId"
    :placeholder="$t('custom-approval.approval-flow.node.select-approver')"
    :consistent-menu-width="false"
    :disabled="!allowAdmin"
    @update:value="handleUpdate"
  />
</template>

<script lang="ts" setup>
import { type SelectOption, type SelectGroupOption, NSelect } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoleStore, useSettingV1Store } from "@/store";
import { isCustomRole } from "@/types";
import {
  ApprovalNode,
  ApprovalNode_GroupValue,
  ApprovalNode_Type,
} from "@/types/proto/v1/issue_service";
import { ExternalApprovalSetting } from "@/types/proto/v1/setting_service";
import { approvalNodeGroupValueText, approvalNodeRoleText } from "@/utils";
import { useCustomApprovalContext } from "../context";

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
const settingStore = useSettingV1Store();
const context = useCustomApprovalContext();
const { allowAdmin } = context;
const { roleList } = storeToRefs(useRoleStore());

const settingValue = computed(() => {
  const setting = settingStore.getSettingByName(
    "bb.workspace.approval.external"
  );
  return (
    setting?.value?.externalApprovalSettingValue ??
    ExternalApprovalSetting.fromJSON({})
  );
});

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

  const externalApprovalNodes =
    settingValue.value.nodes.map<ApprovalNodeSelectOption>((node) => ({
      node: {
        type: ApprovalNode_Type.ANY_IN_GROUP,
        externalNodeId: node.id,
      },
      label: node.title,
      value: node.id,
    }));

  if (customRoleNodes.length === 0 && externalApprovalNodes.length === 0) {
    return presetGroupValueNodes;
  }

  const system: SelectGroupOption = {
    type: "group",
    label: t("custom-approval.approval-flow.node.roles.system"),
    key: "system",
    children: presetGroupValueNodes,
  };
  const groups = [system];
  if (customRoleNodes.length > 0) {
    const custom: SelectGroupOption = {
      type: "group",
      label: t("custom-approval.approval-flow.node.roles.custom"),
      key: "custom",
      children: customRoleNodes,
    };
    groups.push(custom);
  }
  if (externalApprovalNodes.length > 0) {
    const external: SelectGroupOption = {
      type: "group",
      label: t("custom-approval.approval-flow.external-approval.self"),
      children: externalApprovalNodes,
    };
    groups.push(external);
  }

  return groups;
});

const handleUpdate = (
  value: ApprovalNodeSelectOption["value"],
  option: SelectOption
) => {
  const { node } = option as ApprovalNodeSelectOption;
  emit("update:value", node);
};
</script>
