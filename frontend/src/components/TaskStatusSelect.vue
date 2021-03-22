<template>
  <BBSelect
    :selectedItem="selectedStatus"
    :itemList="['OPEN', 'DONE', 'CANCELED']"
    :placeholder="'Unknown Status'"
    :disabled="disabled"
    @select-item="(status, didSelect) => changeStatus(status, didSelect)"
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
import { TaskStatus, TaskStatusTransitionType } from "../types";

export default {
  name: "TaskStatusSelect",
  emits: ["start-transition"],
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
  setup(_, { emit }) {
    const changeStatus = (newStatus: TaskStatus, didChange: () => {}) => {
      let transition: TaskStatusTransitionType;
      switch (newStatus) {
        case "OPEN":
          transition = "REOPEN";
          break;
        case "DONE":
          transition = "RESOLVE";
          break;
        case "CANCELED":
          transition = "ABORT";
          break;
      }
      emit("start-transition", transition, didChange);
    };

    return { changeStatus };
  },
};
</script>
