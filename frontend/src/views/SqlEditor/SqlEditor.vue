<template>
  <div class="sqleditor--wrapper">
    <QuerySelector />
    <Splitpanes class="default-theme splitpanes-wrap">
      <Pane size="20" min-size="0" max-size="30">
        <AsidePanel />
      </Pane>
      <Pane size="80">
        <template v-if="hasTabs">
          <Splitpanes horizontal class="default-theme">
            <Pane size="60">
              <EditorPanel :key="paneKey" />
            </Pane>
            <Pane size="40">
              <TablePanel />
            </Pane>
          </Splitpanes>
        </template>
        <template v-else>
          <GettingStarted />
        </template>
      </Pane>
    </Splitpanes>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useStore } from "vuex";
import AsidePanel from "./AsidePanel/AsidePanel.vue";
import EditorPanel from "./EditorPanel/EditorPanel.vue";
import QuerySelector from "./QuerySelector.vue";
import TablePanel from "./TablePanel/TablePanel.vue";
import GettingStarted from "./GettingStarted.vue";

const store = useStore();

const hasTabs = computed(() => store.getters["editorSelector/hasTabs"]);
const paneKey = computed(() => store.getters["editorSelector/currentTab"].id);
</script>

<style>
/* splitpanes pane style */
.splitpanes.default-theme .splitpanes__pane {
  @apply bg-transparent;
}

.splitpanes.default-theme .splitpanes__splitter {
  @apply bg-gray-100;
  min-height: 8px;
  min-width: 8px;
}

.splitpanes.default-theme .splitpanes__splitter:hover {
  @apply bg-indigo-400;
}

.splitpanes.default-theme .splitpanes__splitter::before,
.splitpanes.default-theme .splitpanes__splitter::after {
  @apply bg-gray-700 opacity-50 text-white;
}

.splitpanes.default-theme .splitpanes__splitter:hover::before,
.splitpanes.default-theme .splitpanes__splitter:hover::after {
  @apply bg-white opacity-100;
}
</style>

<style scoped>
.sqleditor--wrapper {
  color: var(--base);
  --base: #444;
  --nav-height: 64px;
  --tab-height: 36px;
  --font-code: "Source Code Pro", monospace;
  --color-branding: #4f46e5;
  --border-color: rgba(200, 200, 200, 0.2);
  height: calc(100vh - var(--nav-height));
}

.splitpanes.default-theme.splitpanes-wrap {
  height: calc(100% - var(--tab-height));
}
</style>
