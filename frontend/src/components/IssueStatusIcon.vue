<template>
  <span
    class="flex items-center justify-center rounded-full select-none"
    :class="issueIconClass()"
  >
    <template v-if="issueStatus === `OPEN`">
      <span
        v-if="taskStatus === 'RUNNING'"
        class="h-2 w-2 bg-info rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      ></span>
      <span
        v-else-if="taskStatus === 'FAILED'"
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
      <span
        v-else
        class="h-1.5 w-1.5 bg-info rounded-full"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="issueStatus === `DONE`">
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
    <template v-else-if="issueStatus === `CANCELED`">
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
import { TaskStatus, IssueStatus } from "../types";

type SizeType = "small" | "normal";

export default {
  name: "IssueStatusIcon",
  props: {
    issueStatus: {
      required: true,
      type: String as PropType<IssueStatus>,
    },
    // Specify taskStatus if we want to show the task specific status when issueStatus is OPEN.
    taskStatus: {
      type: String as PropType<TaskStatus>,
    },
    size: {
      type: String as PropType<SizeType>,
      default: "normal",
    },
  },
  setup(props) {
    const issueIconClass = () => {
      let iconClass = props.size === "normal" ? "w-5 h-5" : "w-4 h-4";
      switch (props.issueStatus) {
        case "OPEN":
          if (props.taskStatus && props.taskStatus === "FAILED") {
            return iconClass + " bg-error text-white";
          }
          return iconClass + " bg-white border-2 border-info text-info";
        case "CANCELED":
          return iconClass + " bg-white border-2 text-gray-400 border-gray-400";
        case "DONE":
          return iconClass + " bg-success text-white";
      }
    };

    return {
      issueIconClass,
    };
  },
};
</script>
