<template>
  <BBSelect
    :selected-item="selectedRole"
    :item-list="['OWNER', 'DBA', 'DEVELOPER']"
    :placeholder="$t('settings.members.select-role')"
    :disabled="disabled"
    @select-item="(role) => $emit('change-role', role)"
  >
    <template #menuItem="{ item }">
      {{
        $t(`common.role.${item.toLowerCase()}`)
      }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType } from "vue";
import { RoleType } from "../types";

export default {
  name: "RoleSelect",
  props: {
    selectedRole: {
      type: String as PropType<RoleType>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["change-role"],
};
</script>
