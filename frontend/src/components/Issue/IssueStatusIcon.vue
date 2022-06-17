<template>
  <span
    class="flex items-center justify-center rounded-full select-none overflow-hidden"
    :class="issueIconClass()"
  >
    <template v-if="issueStatus === `OPEN`">
      <template v-if="taskStatus === 'RUNNING'">
        <div class="flex h-2 w-2 relative overflow-visible">
          <span
            class="w-full h-full rounded-full z-0 absolute animate-ping-slow"
            style="background-color: rgba(37, 99, 235, 0.5); /* bg-info/50 */"
            aria-hidden="true"
          ></span>
          <span
            class="w-full h-full rounded-full z-[1] bg-info"
            aria-hidden="true"
          ></span>
        </div>
      </template>
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
      <heroicons-solid:check class="w-4 h-4" />
    </template>
    <template v-else-if="issueStatus === `CANCELED`">
      <heroicons-solid:minus class="w-5 h-5" />
    </template>
  </span>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { TaskStatus, IssueStatus } from "../../types";

type SizeType = "small" | "normal";

export default defineComponent({
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
});
</script>
