<template>
  <template v-if="sheet.id !== UNKNOWN_ID">
    <heroicons-outline:user-group
      v-if="sheet.visibility === 'PROJECT'"
      class="w-4 h-4"
    />
    <heroicons-outline:globe
      v-if="sheet.visibility === 'PUBLIC'"
      class="w-4 h-4"
    />
  </template>
  <template v-if="tab.mode === TabMode.Admin">
    <heroicons-outline:wrench class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";

import type { TabInfo } from "@/types";
import { TabMode, unknown, UNKNOWN_ID } from "@/types";
import { useSheetStore } from "@/store";

const props = defineProps({
  tab: {
    type: Object as PropType<TabInfo>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const sheetStore = useSheetStore();

const sheet = computed(() => {
  const { sheetId } = props.tab;
  if (sheetId) {
    const sheet = sheetStore.sheetById.get(sheetId);
    if (sheet) {
      return sheet;
    }
  }
  return unknown("SHEET");
});
</script>
