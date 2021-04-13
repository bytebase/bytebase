<template>
  <BBSelect
    :selectedItem="selectedRole"
    :itemList="['OWNER', 'DEVELOPER']"
    :placeholder="'Select role'"
    :disabled="disabled"
    @select-item="(role) => $emit('change-role', role)"
  >
    <template v-slot:menuItem="{ item }">
      {{ item }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType } from "vue";
import { ProjectRoleType } from "../types";

export default {
  name: "ProjectRoleSelect",
  emits: ["change-role"],
  components: {},
  props: {
    selectedRole: {
      type: String as PropType<ProjectRoleType>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup() {
    return {};
  },
};
</script>
