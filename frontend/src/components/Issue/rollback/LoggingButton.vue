<template>
  <div
    class="select-none inline-flex items-center border border-control-border rounded-md text-control bg-white text-sm leading-5 font-medium focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
    :class="[disabled && 'cursor-not-allowed bg-control-bg opacity-50 ']"
  >
    <span class="pl-3 pr-1">{{ $t("task.rollback.logging") }}</span>
    <BBSpin class="w-4 h-4" />
    <NTooltip :animated="false" :delay="150">
      <template #trigger>
        <span
          class="h-[28px] px-1.5 flex items-center rounded-r-md hover:bg-control-bg-hover cursor-pointer border-l"
          :class="[disabled && 'pointer-events-none']"
          @click="toggleRollback(false)"
        >
          <heroicons-outline:x-mark class="w-3 h-3" />
        </span>
      </template>

      <span>{{ $t("task.rollback.cancel-logging") }}</span>
    </NTooltip>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NTooltip } from "naive-ui";
import { useRollbackLogic } from "./common";

const { allowRollback, toggleRollback } = useRollbackLogic();

const disabled = computed(() => !allowRollback.value);
</script>
