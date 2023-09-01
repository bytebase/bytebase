<template>
  <NTabs
    :v-model:value="tab"
    type="segment"
    size="small"
    class="h-full"
    pane-style="height: calc(100% - 35px); padding: 0;"
  >
    <NTabPane name="INFO" :tab="$t('common.info')">
      <InfoTabPane />
    </NTabPane>
    <NTabPane name="SHEET" :tab="$t('sheet.sheet')">
      <SheetTabPane />
    </NTabPane>
    <NTabPane name="HISTORY" :tab="$t('common.history')">
      <HistoryTabPane />
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import HistoryTabPane from "./HistoryTabPane";
import InfoTabPane from "./InfoTabPane";
import SheetTabPane from "./SheetTabPane";
import { useSecondarySidebarContext } from "./context";

const { tab } = useSecondarySidebarContext();
</script>
