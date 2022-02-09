<template>
  <BBSelect
    :selected-item="selectedRole"
    :item-list="['OWNER', 'DEVELOPER']"
    :placeholder="$t('settings.members.select-role')"
    :disabled="disabled"
    @select-item="(role) => $emit('change-role', role)"
  >
    <template #menuItem="{ item }">
      {{ $t(`common.role.${item.toLowerCase()}`) }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { ProjectRoleType } from "../types";

export default defineComponent({
  name: "ProjectRoleSelect",
  props: {
    selectedRole: {
      type: String as PropType<ProjectRoleType>,
      default: undefined,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["change-role"],
});
</script>
