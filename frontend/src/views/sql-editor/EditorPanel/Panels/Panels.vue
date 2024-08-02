<template>
  <div class="flex-1 flex items-stretch overflow-hidden">
    <GutterBar class="border-r border-control-border" :class="gutterBarClass" />

    <div class="flex-1" :class="contentClass">
      <slot v-if="!viewState || viewState.view === 'CODE'" name="code-panel" />

      <template v-if="viewState">
        <InfoPanel v-if="viewState.view === 'INFO'" />
        <TablesPanel v-if="viewState.view === 'TABLES'" />
        <ViewsPanel v-if="viewState.view === 'VIEWS'" />
        <FunctionsPanel v-if="viewState.view === 'FUNCTIONS'" />
        <ProceduresPanel v-if="viewState.view === 'PROCEDURES'" />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { VueClass } from "@/utils";
import GutterBar from "../GutterBar";
import { useEditorPanelContext } from "../context";
import FunctionsPanel from "./FunctionsPanel";
import InfoPanel from "./InfoPanel";
import ProceduresPanel from "./ProceduresPanel";
import TablesPanel from "./TablesPanel";
import ViewsPanel from "./ViewsPanel";

defineProps<{
  gutterBarClass?: VueClass;
  contentClass?: VueClass;
}>();

const { viewState } = useEditorPanelContext();
</script>
