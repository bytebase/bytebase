<template>
  <div v-if="isCustomRole(role.name)" class="w-full flex justify-end space-x-2">
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
import { computed } from "vue";
import { useCurrentUserV1, useRoleStore } from "@/store";
import { Role } from "@/types/proto/v1/role_service";
import { hasWorkspacePermissionV2, isCustomRole } from "@/utils";
import { useCustomRoleSettingContext } from "../../../context";

const props = defineProps<{
  role: Role;
}>();

defineEmits<{
  (event: "edit", role: Role): void;
}>();

const { hasCustomRoleFeature, showFeatureModal } =
  useCustomRoleSettingContext();
const currentUser = useCurrentUserV1();

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.roles.update");
});

const deleteRole = async () => {
  if (!hasCustomRoleFeature.value) {
    showFeatureModal.value = true;
    return;
  }

  await useRoleStore().deleteRole(props.role);
};
</script>
