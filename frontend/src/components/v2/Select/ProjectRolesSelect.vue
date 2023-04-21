<template>
  <NSelect
    :value="roleList"
    :options="roleOptions"
    :multiple="true"
    :max-tag-count="'responsive'"
    :placeholder="$t('role.select-roles')"
    @update:value="$emit('update:role-list', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { type SelectOption, NSelect } from "naive-ui";

import { useRoleStore } from "@/store";
import { ProjectRoleType } from "@/types";
import { displayRoleTitle } from "@/utils";

defineProps<{
  roleList: ProjectRoleType[];
}>();

defineEmits<{
  (event: "update:role-list", list: ProjectRoleType[]): void;
}>();

const roleOptions = computed(() => {
  return useRoleStore().roleList.map<SelectOption>((role) => {
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
    };
  });
});
</script>
