<template>
  <div class="overflow-hidden">
    <VirtualList
      :items="table.columns"
      :key-field="`name`"
      :item-resizable="true"
      :item-size="24"
    >
      <template #default="{ item: column }: VirtualListItem">
        <div class="flex items-start px-4">
          <div
            class="text-sm leading-6 text-gray-600whitespace-pre-wrap break-words flex-2 min-w-[4rem]"
          >
            {{ column.name }}
          </div>
          <div
            class="shrink-0 text-right text-sm leading-6 text-gray-400 overflow-x-hidden whitespace-nowrap flex-1 min-w-[4rem]"
          >
            {{ column.type }}
          </div>
        </div>
      </template>
    </VirtualList>
  </div>
</template>

<script setup lang="ts">
import { VirtualList } from "vueuc";
import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

export type VirtualListItem = {
  item: ColumnMetadata;
  index: number;
};

defineProps<{
  table: TableMetadata;
}>();
</script>
