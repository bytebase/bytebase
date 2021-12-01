<template>
  <span
    class="flex items-center justify-center rounded-full select-none"
    :class="iconClass()"
  >
    <template v-if="status === `PENDING`">
      <span
        class="h-2 w-2 bg-info rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="status === `DONE`">
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
    <template v-else-if="status === `FAILED`">
      <span
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
  </span>
</template>

<script lang="ts">
import { PropType } from "vue";
import { MigrationStatus } from "../types";

export default {
  name: "IssueStatusIcon",
  props: {
    status: {
      required: true,
      type: String as PropType<MigrationStatus>,
    },
  },
  setup(props) {
    const iconClass = () => {
      let iconClass = "w-5 h-5";
      switch (props.status) {
        case "PENDING":
          return iconClass + " bg-white border-2 border-info text-info";
        case "DONE":
          return iconClass + " bg-success text-white";
        case "FAILED":
          return iconClass + " bg-error text-white";
      }
    };

    return {
      iconClass,
    };
  },
};
</script>
