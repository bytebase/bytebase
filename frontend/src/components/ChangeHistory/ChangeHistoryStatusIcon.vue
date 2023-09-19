<template>
  <span
    class="flex items-center justify-center rounded-full select-none"
    :class="iconClass()"
  >
    <template v-if="status === ChangeHistory_Status.PENDING">
      <span
        class="h-2 w-2 bg-info rounded-full"
        style="animation: pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite"
        aria-hidden="true"
      ></span>
    </template>
    <template v-else-if="status === ChangeHistory_Status.DONE">
      <heroicons-solid:check class="w-4 h-4" />
    </template>
    <template v-else-if="status === ChangeHistory_Status.FAILED">
      <span
        class="h-2 w-2 rounded-full text-center pb-6 font-normal text-base"
        aria-hidden="true"
        >!</span
      >
    </template>
  </span>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { ChangeHistory_Status } from "@/types/proto/v1/database_service";

const props = defineProps({
  status: {
    required: true,
    type: Number as PropType<ChangeHistory_Status>,
  },
});

const iconClass = () => {
  const iconClass = "w-5 h-5";
  switch (props.status) {
    case ChangeHistory_Status.PENDING:
      return iconClass + " bg-white border-2 border-info text-info";
    case ChangeHistory_Status.DONE:
      return iconClass + " bg-success text-white";
    case ChangeHistory_Status.FAILED:
      return iconClass + " bg-error text-white";
  }
};
</script>
