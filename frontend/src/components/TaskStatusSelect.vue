<template>
  <select
    id="status"
    name="status"
    class="pl-3 pr-10 py-2 btn-select w-full"
    @change="
      (e) => {
        $emit('change-status', e.target.value);
      }
    "
  >
    <option value="OPEN" :selected="selectedStatus == 'OPEN'">Open</option>
    <option value="DONE" :selected="selectedStatus == 'DONE'">Done</option>
    <option value="CANCELED" :selected="selectedStatus == 'CANCELED'">
      Canceled
    </option>
  </select>
</template>

<script lang="ts">
import { PropType } from "vue";
import { TaskStatus } from "../types";

export default {
  name: "TaskStatusSelect",
  emits: ["change-status"],
  components: {},
  props: {
    selectedStatus: {
      type: String as PropType<TaskStatus>,
    },
  },
  setup() {},
};
</script>
