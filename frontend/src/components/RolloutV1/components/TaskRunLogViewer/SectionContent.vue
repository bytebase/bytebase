<template>
  <NVirtualList
    :items="section.items"
    :item-size="ITEM_HEIGHT"
    item-resizable
    :style="{ maxHeight: `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px` }"
    class="bg-gray-50 border-t border-gray-100"
  >
    <template #default="{ item, index }">
      <div
        class="flex items-start gap-x-2 py-0.5 hover:bg-gray-100"
        :class="[indent ? 'px-6' : 'px-3', { 'border-t border-gray-100': index > 0 }]"
      >
        <span class="text-gray-300 w-6 text-right shrink-0 tabular-nums">
          {{ index + 1 }}
        </span>
        <span class="text-gray-400 shrink-0 tabular-nums">
          {{ item.time }}
        </span>
        <span
          v-if="item.relativeTime"
          class="text-gray-300 shrink-0 tabular-nums"
        >
          {{ item.relativeTime }}
        </span>
        <span :class="item.levelClass" class="shrink-0">
          {{ item.levelIndicator }}
        </span>
        <span :class="item.detailClass" class="break-all">
          {{ item.detail }}
        </span>
        <span
          v-if="item.affectedRows !== undefined"
          class="text-gray-400 shrink-0 ml-auto"
        >
          {{ item.affectedRows }} rows
        </span>
      </div>
    </template>
  </NVirtualList>
</template>

<script lang="ts" setup>
import { NVirtualList } from "naive-ui";
import type { Section } from "./types";

const ITEM_HEIGHT = 20;
const MAX_VISIBLE_ITEMS = 10;

defineProps<{
  section: Section;
  indent?: boolean;
}>();
</script>
