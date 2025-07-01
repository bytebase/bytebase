<template>
  <SheetConnectionIcon :tab="tab" class="w-4 h-4" />
  <heroicons-outline:user-group
    v-if="
      sheet?.visibility === Worksheet_Visibility.PROJECT_READ ||
      sheet?.visibility === Worksheet_Visibility.PROJECT_WRITE
    "
    class="w-4 h-4"
  />
  <template v-if="tab.mode === 'ADMIN'">
    <WrenchIcon class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import { WrenchIcon } from "lucide-vue-next";
import { computed } from "vue";
import { useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import { SheetConnectionIcon } from "../../EditorCommon";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const sheetV1Store = useWorkSheetStore();

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
