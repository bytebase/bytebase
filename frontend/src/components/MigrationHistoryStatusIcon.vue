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
      <heroicons-solid:check class="w-4 h-4" />
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
import { defineComponent, PropType } from "vue";
import { MigrationStatus } from "../types";

export default defineComponent({
  name: "MigrationHistoryStatusIcon",
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
});
</script>
