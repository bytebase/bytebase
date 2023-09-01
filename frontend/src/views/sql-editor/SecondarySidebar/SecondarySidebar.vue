<template>
  <NTabs
    :v-model:value="tab"
    type="segment"
    size="small"
    class="h-full"
    pane-style="height: calc(100% - 35px); padding: 0;"
  >
    <NTabPane name="INFO" :tab="$t('common.info')">
      <InfoPanel />
    </NTabPane>
    <NTabPane name="SHEET" :tab="$t('sheet.sheet')"> Sheet </NTabPane>
    <NTabPane name="HISTORY" :tab="$t('common.history')">
      <HistoryPanel />
    </NTabPane>
    <!-- <NTabPane name="projects" :tab="$t('common.projects')">
      <Splitpanes horizontal class="default-theme" :dbl-click-splitter="false">
        <Pane>
          <DatabaseTree
            key="sql-editor-database-tree"
            v-model:search-pattern="searchPattern"
            @alter-schema="$emit('alter-schema', $event)"
          />
        </Pane>
        <Pane v-if="showSchemaPanel" :size="40">
          <SchemaPanel @alter-schema="$emit('alter-schema', $event)" />
        </Pane>
      </Splitpanes>
    </NTabPane>
    <NTabPane
      v-if="hasInstanceView"
      name="instances"
      :tab="$t('common.instances')"
    >
      <Splitpanes horizontal class="default-theme" :dbl-click-splitter="false">
        <Pane>
          <DatabaseTree
            key="sql-editor-database-tree"
            v-model:search-pattern="searchPattern"
            @alter-schema="$emit('alter-schema', $event)"
          />
        </Pane>
        <Pane v-if="showSchemaPanel" :size="40">
          <SchemaPanel @alter-schema="$emit('alter-schema', $event)" />
        </Pane>
      </Splitpanes>
    </NTabPane>
    <NTabPane name="history" :tab="$t('common.history')">
      <QueryHistoryContainer />
    </NTabPane> -->
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import HistoryPanel from "./HistoryPanel";
import InfoPanel from "./InfoPanel";
import { useSecondarySidebarContext } from "./context";

const { tab } = useSecondarySidebarContext();
</script>
