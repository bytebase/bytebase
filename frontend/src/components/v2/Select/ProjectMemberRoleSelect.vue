<template>
  <NSelect
    :value="role"
    :options="roleOptions"
    :max-tag-count="'responsive'"
    :placeholder="'Select role'"
    @update:value="$emit('update:role', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { type SelectOption, NSelect } from "naive-ui";

import { featureToRef, useRoleStore } from "@/store";
import { PresetRoleType, ProjectRoleType } from "@/types";
import { displayRoleTitle } from "@/utils";

defineProps<{
  role: ProjectRoleType;
}>();

defineEmits<{
  (event: "update:role", role: ProjectRoleType): void;
}>();

const hasCustomRoleFeature = featureToRef("bb.feature.custom-role");

const roleOptions = computed(() => {
  let roleList = useRoleStore().roleList;
  // For enterprise plan, we don't allow to add exporter role.
  if (hasCustomRoleFeature.value) {
    roleList = useRoleStore().roleList.filter((role) => {
      return role.name !== PresetRoleType.EXPORTER;
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
