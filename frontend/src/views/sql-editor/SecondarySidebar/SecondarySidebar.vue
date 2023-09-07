<template>
  <div class="h-full">
    <InfoTabPane
      v-if="tab === 'INFO'"
      @alter-schema="$emit('alter-schema', $event)"
    />
    <SheetTabPane v-if="tab === 'SHEET'" />
    <HistoryTabPane v-if="tab === 'HISTORY'" />
  </div>
</template>

<script setup lang="ts">
import HistoryTabPane from "./HistoryTabPane";
import InfoTabPane from "./InfoTabPane";
import SheetTabPane from "./SheetTabPane";
import { useSecondarySidebarContext } from "./context";

defineEmits<{
  (
    event: "alter-schema",
    params: { databaseId: string; schema: string; table: string }
  ): void;
}>();

const { tab } = useSecondarySidebarContext();
</script>
