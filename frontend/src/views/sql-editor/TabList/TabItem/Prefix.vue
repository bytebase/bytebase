<template>
  <SheetConnectionIcon :tab="tab" class="w-4 h-4" />

  <template v-if="sheet">
    <heroicons-outline:user-group
      v-if="sheet.visibility === Sheet_Visibility.VISIBILITY_PROJECT"
      class="w-4 h-4"
    />
    <heroicons-outline:globe
      v-if="sheet.visibility === Sheet_Visibility.VISIBILITY_PUBLIC"
      class="w-4 h-4"
    />
  </template>
  <template v-if="tab.mode === TabMode.Admin">
    <heroicons-outline:wrench class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useSheetV1Store } from "@/store";
import type { TabInfo } from "@/types";
import { TabMode } from "@/types";
import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";
import { SheetConnectionIcon } from "../../EditorCommon";

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

const sheetV1Store = useSheetV1Store();

const sheet = computed(() => {
  const { sheetName } = props.tab;
  if (sheetName) {
    const sheet = sheetV1Store.getSheetByName(sheetName);
    if (sheet) {
      return sheet;
    }
  }
  return null;
});
</script>
