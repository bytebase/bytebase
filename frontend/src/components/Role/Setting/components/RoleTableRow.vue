<template>
  <div class="bb-grid-cell whitespace-nowrap">
    {{ displayRoleTitle(role.name) }}
    <SystemLabel v-if="!isCustomRole(props.role.name)" class="ml-1" />
  </div>

  <div class="bb-grid-cell">
    {{ displayRoleDescription(role.name) }}
  </div>
  <div class="bb-grid-cell gap-x-1">
    <template v-if="isCustomRole(props.role.name)">
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
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import SystemLabel from "@/components/SystemLabel.vue";
import { SpinnerButton } from "@/components/v2";
import { useRoleStore } from "@/store";
import { isCustomRole } from "@/types";
import type { Role } from "@/types/proto/v1/role_service";
import {
  displayRoleDescription,
  displayRoleTitle,
  useWorkspacePermissionV1,
} from "@/utils";
import { useCustomRoleSettingContext } from "../context";

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
