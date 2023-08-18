<template>
  <div
    :id="domIDForItem(item)"
    class="flex items-start justify-between hover:bg-gray-100 px-2 gap-x-1"
    :class="[isCurrentItem && 'bg-indigo-600/10']"
    @click="$emit('click', item, $event)"
    @contextmenu="$emit('contextmenu', item, $event)"
  >
    <SheetConnectionIcon :sheet="item.target" class="shrink-0 w-4 h-6" />
    <div class="flex-1 text-sm leading-6 cursor-pointer break-all">
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span v-if="item.target.title" v-html="titleHTML(item, keyword)" />
      <span v-else>
        {{ $t("sql-editor.untitled-sheet") }}
      </span>
    </div>

    <div
      v-if="unsaved"
      class="shrink-0 w-4 h-6 flex items-center justify-center"
      @click.stop
    >
      <NTooltip>
        <template #trigger>
          <carbon:dot-mark class="text-gray-500 w-4 h-4" />
        </template>
        <template #default>
          <span>{{ $t("sql-editor.tab.unsaved") }}</span>
        </template>
      </NTooltip>
    </div>
    <div class="shrink-0" @click.stop>
      <Dropdown :sheet="item.target" :view="view" :secondary="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useTabStore } from "@/store";
import { Dropdown } from "@/views/sql-editor/Sheet";
import { SheetViewMode } from "@/views/sql-editor/Sheet";
import { SheetConnectionIcon } from "../../EditorCommon";
import { MergedItem, SheetItem, domIDForItem, titleHTML } from "./common";

const props = defineProps<{
  item: SheetItem;
  isCurrentItem: boolean;
  view: SheetViewMode;
  keyword: string;
}>();

defineEmits<{
  (event: "click", item: MergedItem, e: MouseEvent): void;
  (event: "contextmenu", item: MergedItem, e: MouseEvent): void;
}>();

const tabStore = useTabStore();

const unsaved = computed(() => {
  const tab = tabStore.tabList.find(
    (tab) => tab.sheetName === props.item.target.name
  );
  if (tab) {
    return !tab.isSaved;
  }
  console.assert(false, "should never reach this line");
  return false;
});
</script>
