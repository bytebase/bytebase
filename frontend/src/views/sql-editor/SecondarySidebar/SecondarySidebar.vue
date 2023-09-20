<template>
  <div class="h-full">
    <InfoTabPane v-if="tab === 'INFO'" />
    <SheetTabPane v-if="tab === 'SHEET'" />
    <HistoryTabPane v-if="tab === 'HISTORY'" />
  </div>
</template>

<script setup lang="ts">
import HistoryTabPane from "./HistoryTabPane";
import InfoTabPane from "./InfoTabPane";
import SheetTabPane from "./SheetTabPane";
import { useSecondarySidebarContext } from "./context";

const { tab } = useSecondarySidebarContext();
</script>
