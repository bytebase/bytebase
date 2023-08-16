<template>
  <div
    :id="domIDForItem(item)"
    class="flex items-start justify-between hover:bg-gray-100 px-2 gap-x-1"
    :class="[isCurrentItem && 'bg-indigo-600/10']"
    @click="$emit('click', item, $event)"
    @contextmenu="$emit('contextmenu', item, $event)"
  >
    <div class="flex-1 text-sm cursor-pointer pt-0.5 break-all">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-if="item.target.title" v-html="titleHTML(item, keyword)" />
      <span v-else>
        {{ $t("sql-editor.untitled-sheet") }}
      </span>
    </div>
    <div class="shrink-0" @click.stop>
      <Dropdown :sheet="item.target" :view="view" :secondary="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Dropdown } from "@/views/sql-editor/Sheet";
import { SheetViewMode } from "@/views/sql-editor/Sheet";
import { MergedItem, SheetItem, domIDForItem, titleHTML } from "./common";

defineProps<{
  item: SheetItem;
  isCurrentItem: boolean;
  view: SheetViewMode;
  keyword: string;
}>();

defineEmits<{
  (event: "click", item: MergedItem, e: MouseEvent): void;
  (event: "contextmenu", item: MergedItem, e: MouseEvent): void;
}>();
</script>
