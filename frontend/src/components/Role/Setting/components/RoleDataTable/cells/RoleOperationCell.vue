<template>
  <div v-if="isCustomRole(role.name)" class="space-x-2">
    <NButton size="tiny" :disabled="!allowAdmin" @click="$emit('edit', role)">
      {{ $t("common.edit") }}
    </NButton>
    <SpinnerButton
      size="tiny"
      :disabled="!allowAdmin"
      :tooltip="$t('role.setting.delete')"
      :on-confirm="deleteRole"
    >
      {{ $t("common.delete") }}
    </SpinnerButton>
  </div>
</template>

<script lang="ts" setup>
import { useRoleStore } from "@/store";
import { Role } from "@/types/proto/v1/role_service";
import { isCustomRole, useWorkspacePermissionV1 } from "@/utils";
import { useCustomRoleSettingContext } from "../../../context";

const props = defineProps<{
  role: Role;
}>();

defineEmits<{
  (event: "edit", role: Role): void;
}>();

const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();

const allowAdmin = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-general"
);

const deleteRole = async () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  await useRoleStore().deleteRole(props.role);
};
</script>
