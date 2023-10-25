<template>
  <TextOverflowPopover
    :content="statement"
    :max-length="100"
    :max-popover-content-length="1000"
    :line-wrap="false"
    content-class="line-clamp-1 flex-1"
    placement="bottom"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { useSheetV1Store } from "@/store";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { getSheetStatement } from "@/utils";

const props = defineProps<{
  change: Change;
}>();

const sheet = computed(() => {
  return useSheetV1Store().getSheetByName(props.change.sheet);
});

const statement = computed(() => {
  if (!sheet.value) return "";
  return getSheetStatement(sheet.value);
});
</script>
