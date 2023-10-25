<template>
  <TextOverflowPopover
    :content="content"
    :max-length="100"
    placement="bottom"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { getAffectedTablesOfChangeHistory } from "@/utils";
import { getAffectedTableDisplayName } from "../utils";

const props = defineProps<{
  changeHistory: ChangeHistory;
}>();

const content = computed(() => {
  return getAffectedTablesOfChangeHistory(props.changeHistory)
    .map(getAffectedTableDisplayName)
    .join(", ");
});
</script>
