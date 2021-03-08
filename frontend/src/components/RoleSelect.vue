<template>
  <BBSelect
    :selectedItem="selectedRole"
    :itemList="['OWNER', 'DBA', 'DEVELOPER']"
    :placeholder="'Select role'"
    @select-item="(role) => $emit('change-role', role)"
  >
    <template v-slot:menuItem="{ item }">
      {{ item }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType } from "vue";
import { RoleType } from "../types";

export default {
  name: "RoleSelect",
  emits: ["change-role"],
  components: {},
  props: {
    selectedRole: {
      type: String as PropType<RoleType>,
    },
  },
  setup() {
    return {};
  },
};
</script>
