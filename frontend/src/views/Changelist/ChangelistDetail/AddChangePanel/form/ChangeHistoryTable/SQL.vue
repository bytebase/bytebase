<template>
  <TextOverflowPopover
    :content="changeHistory.statement"
    :max-length="100"
    placement="bottom"
  />
</template>

<script setup lang="ts">
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { ChangeHistory } from "@/types/proto/v1/database_service";

defineProps<{
  changeHistory: ChangeHistory;
}>();
</script>
