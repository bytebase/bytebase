<template>
  <NPopover>
    <template #trigger>
      <div v-bind="$attrs" class="!no-underline">
        <span class="!no-underline">(</span>
        <span class="!no-underline">{{ summary.done }}</span>
        <span class="!no-underline">/</span>
        <span class="!no-underline">{{ summary.failed }}</span>
        <span class="!no-underline mx-1">&amp;</span>
        <span class="!no-underline">{{ summary.canceled }}</span>
        <span class="!no-underline">/</span>
        <span class="!no-underline">{{ summary.running }}</span>
        <span class="!no-underline">/</span>
        <span class="!no-underline">{{ summary.total }}</span>
        <span class="!no-underline">)</span>
      </div>
    </template>

    <div class="w-28 flex flex-col gap-y-0.5">
      <div class="flex justify-between">
        <label class="textlabel">{{ $t("task.status.success") }}</label>
        <span>{{ summary.done }}</span>
      </div>
      <div class="flex justify-between">
        <label class="textlabel">{{ $t("task.status.failed") }}</label>
        <span>{{ summary.failed }}</span>
      </div>
      <hr class="my-0.5" />
      <div class="flex justify-between">
        <label class="textlabel">{{ $t("task.status.running") }}</label>
        <span>{{ summary.running }}</span>
      </div>
      <div class="flex justify-between">
        <label class="textlabel">{{ $t("task.status.canceled") }}</label>
        <span>{{ summary.canceled }}</span>
      </div>
      <div class="flex justify-between">
        <label class="textlabel">{{ $t("common.total") }}</label>
        <span>{{ summary.total }}</span>
      </div>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { NPopover } from "naive-ui";

import type { Stage } from "@/types";

const props = defineProps<{
  stage: Stage;
}>();

const summary = computed(() => {
  const { taskList } = props.stage;
  const summary = {
    done: 0,
    failed: 0,
    canceled: 0,
    running: 0,
    total: taskList.length,
  };
  taskList.forEach((task) => {
    switch (task.status) {
      case "DONE":
        summary.done++;
        break;
      case "FAILED":
        summary.failed++;
        break;
      case "CANCELED":
        summary.canceled++;
        break;
      case "RUNNING":
        summary.running++;
        break;
    }
  });
  return summary;
});
</script>
