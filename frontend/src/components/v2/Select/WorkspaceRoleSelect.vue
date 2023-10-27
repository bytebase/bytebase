<template>
  <NSelect
    :value="role"
    :options="roleOptions"
    :filterable="true"
    :filter="filterByTitle"
    :placeholder="$t('role.select-role')"
    class="bb-workspace-role-select"
    @update:value="$emit('update:role', $event)"
  />
</template>

<script lang="ts" setup>
import { type SelectOption, NSelect } from "naive-ui";
import { computed } from "vue";
import { UserRole } from "@/types/proto/v1/auth_service";
import { roleNameV1 } from "@/utils";

defineProps<{
  role?: UserRole;
}>();

defineEmits<{
  (event: "update:role", role: UserRole): void;
}>();

const roleOptions = computed(() => {
  const roleList = [UserRole.OWNER, UserRole.DBA, UserRole.DEVELOPER];
  return roleList.map<SelectOption>((role) => ({
    label: roleNameV1(role),
    value: role,
  }));
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { value } = option as SelectOption;
  pattern = pattern.toLowerCase();
  return String(value).toLowerCase().includes(pattern);
};
</script>
