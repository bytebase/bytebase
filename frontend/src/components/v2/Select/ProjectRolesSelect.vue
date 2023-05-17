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

import { featureToRef, useRoleStore } from "@/store";
import { PresetRoleType, ProjectRoleType } from "@/types";
import { displayRoleTitle } from "@/utils";

defineProps<{
  roleList: ProjectRoleType[];
}>();

defineEmits<{
  (event: "update:role-list", list: ProjectRoleType[]): void;
}>();

const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");

const roleOptions = computed(() => {
  let roleList = useRoleStore().roleList;
  if (hasCustomRoleFeature.value) {
    roleList = useRoleStore().roleList.filter((role) => {
      return (
        role.name !== PresetRoleType.Exporter &&
        role.name !== PresetRoleType.Querier
      );
    });
  }
  return roleList.map<SelectOption>((role) => {
    return {
      label: displayRoleTitle(role.name),
      value: role.name,
    };
  });
});
</script>
