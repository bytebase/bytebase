<template>
  <div class="bb-grid-cell whitespace-nowrap">
    {{ extractRoleResourceName(role.name) }}
  </div>

  <div class="bb-grid-cell">
    {{ role.description }}
  </div>
  <div class="bb-grid-cell gap-x-1">
    <template v-if="allowEdit">
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
import { computed } from "vue";
import { NButton } from "naive-ui";

import type { Role } from "@/types/proto/v1/role_service";
import { extractRoleResourceName, useWorkspacePermission } from "@/utils";
import { SpinnerButton } from "@/components/v2";
import { useRoleStore } from "@/store";

const props = defineProps<{
  role: Role;
}>();

defineEmits<{
  (event: "edit", role: Role): void;
}>();

const allowAdmin = useWorkspacePermission(
  "bb.permission.workspace.manage-general"
);

const allowEdit = computed(() => {
  return (
    props.role.name !== "roles/OWNER" && props.role.name !== "roles/DEVELOPER"
  );
});

const deleteRole = async () => {
  await useRoleStore().deleteRole(props.role);
};
</script>
