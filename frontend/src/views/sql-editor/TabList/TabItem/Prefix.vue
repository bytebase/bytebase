<template>
  <SheetConnectionIcon :tab="tab" class="w-4 h-4" />
  <heroicons-outline:user-group
    v-if="
      sheet?.visibility === Worksheet_Visibility.VISIBILITY_PROJECT_READ ||
      sheet?.visibility === Worksheet_Visibility.VISIBILITY_PROJECT_WRITE
    "
    class="w-4 h-4"
  />
  <template v-if="tab.mode === 'ADMIN'">
    <heroicons-outline:wrench class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
import { useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { Worksheet_Visibility } from "@/types/proto/v1/worksheet_service";
import { SheetConnectionIcon } from "../../EditorCommon";

const props = defineProps({
  tab: {
    type: Object as PropType<SQLEditorTab>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const sheetV1Store = useWorkSheetStore();

const sheet = computed(() => {
  const { sheet: sheetName } = props.tab;
  if (sheetName) {
    const sheet = sheetV1Store.getSheetByName(sheetName);
    if (sheet) {
      return sheet;
    }
  }
  return null;
});
</script>
