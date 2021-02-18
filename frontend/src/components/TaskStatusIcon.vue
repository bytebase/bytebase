<template>
  <span
    class="w-5 h-5 flex items-center justify-center rounded-full select-none"
    :class="taskIconClass(task)"
  >
    <template v-if="task.attributes.status == `OPEN`">
      <span
        v-if="activeStage(task).status == 'RUNNING'"
        class="h-2 w-2 bg-blue-600 hover:bg-blue-700 rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      ></span>
      <span
        v-else-if="activeStage(task).status == 'FAILED'"
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
      <span
        v-else
        class="h-1.5 w-1.5 bg-blue-600 hover:bg-blue-700 rounded-full"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="task.attributes.status == `DONE`">
      <svg
        class="w-4 h-4"
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          fill-rule="evenodd"
          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
          clip-rule="evenodd"
        />
      </svg>
    </template>
    <template v-else-if="task.attributes.status == `CANCELED`">
      <svg
        class="w-5 h-5"
        fill="currentColor"
        viewBox="0 0 20 20"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        >
        <path
          fill-rule="evenodd"
          d="M3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
          clip-rule="evenodd"
        ></path>
      </svg>
    </template>
  </span>
</template>

<script lang="ts">
import { PropType } from "vue";
import { activeStage } from "../utils";
import { Task } from "../types";

export default {
  name: "TaskStatusIcon",
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
  },
  components: {},
  setup(props, ctx) {
    const taskIconClass = (task: Task) => {
      switch (task.attributes.status) {
        case "OPEN":
          switch (activeStage(task).status) {
            case "FAILED":
              return "bg-error text-white hover:text-white hover:bg-error-hover";
            case "RUNNING":
            case "PENDING":
            case "DONE":
            case "SKIPPED":
              return "bg-white border-2 border-blue-600 text-blue-600 hover:text-blue-700 hover:border-blue-700";
          }
        case "CANCELED":
          return "bg-white border-2 text-gray-400 border-gray-400 hover:text-gray-500 hover:border-gray-500";
        case "DONE":
          return "bg-success hover:bg-success-hover text-white";
      }
    };

    return {
      taskIconClass,
      activeStage,
    };
  },
};
</script>
