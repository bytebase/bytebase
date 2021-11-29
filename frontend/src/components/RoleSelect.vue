<template>
  <BBSelect
    :selected-item="selectedRole"
    :item-list="['OWNER', 'DBA', 'DEVELOPER']"
    :placeholder="'Select role'"
    :disabled="disabled"
    @select-item="(role) => $emit('change-role', role)"
  >
    <template #menuItem="{ item }">
      {{
        item == "DBA"
          ? item
          : item.charAt(0).toUpperCase() + item.toLowerCase().slice(1)
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
