<template>
  <BBSelect
    :selectedItem="selectedStatus"
    :itemList="['OPEN', 'DONE', 'CANCELED']"
    :placeholder="'Unknown Status'"
    :disabled="disabled"
    @select-item="(status) => $emit('change-status', status)"
  >
    <template v-slot:menuItem="{ item }">
      <span class="flex items-center space-x-2">
        <TaskStatusIcon :taskStatus="item" :size="'small'" />
        <span>
          {{ item }}
        </span>
      </span>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType } from "vue";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import { TaskStatus } from "../types";

export default {
  name: "TaskStatusSelect",
  emits: ["change-status"],
  components: { TaskStatusIcon },
  props: {
    selectedStatus: {
      type: String as PropType<TaskStatus>,
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
