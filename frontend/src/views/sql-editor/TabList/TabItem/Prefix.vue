<template>
  <div class="opacity-80 flex items-center gap-x-2">
    <PencilLineIcon v-if="isDraft" class="w-4 h-4" />
    <template v-else>
      <UsersIcon
        v-if="sheet && !isWorksheetCreator(sheet)"
        class="w-4 h-4"
      />
      <WrenchIcon v-if="tab.mode === 'ADMIN'" class="w-4 h-4" />
    </template>
    <SheetConnectionIcon :tab="tab" class="w-4 h-4" />
  </div>
</template>

<script lang="ts" setup>
import { PencilLineIcon, UsersIcon, WrenchIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { useSheetContext } from "@/views/sql-editor/Sheet";
import { SheetConnectionIcon } from "../../EditorCommon";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const sheetV1Store = useWorkSheetStore();
const { isWorksheetCreator } = useSheetContext();

const isDraft = computed(() => {
  const { worksheet, viewState } = props.tab;
  if (worksheet) {
    return false;
  }
  return viewState.view === "CODE";
});

const sheet = computed(() => {
  const { worksheet: sheetName } = props.tab;
  if (sheetName) {
    const sheet = sheetV1Store.getWorksheetByName(sheetName);
    if (sheet) {
      return sheet;
    }
  }
  return null;
});
</script>
