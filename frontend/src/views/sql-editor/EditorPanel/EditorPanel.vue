<template>
  <div class="w-full flex-1 flex flex-row items-stretch overflow-hidden">
    <Panels>
      <template #code-panel>
        <StandardPanel v-if="!currentTab || currentTab.mode === 'WORKSHEET'" />

        <TerminalPanel v-else-if="currentTab.mode === 'ADMIN'" />

        <AccessDenied v-else />
      </template>
    </Panels>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import { useSQLEditorTabStore } from "@/store";
import AccessDenied from "./AccessDenied.vue";
import Panels from "./Panels";
import StandardPanel from "./StandardPanel";
import TerminalPanel from "./TerminalPanel";

const tabStore = useSQLEditorTabStore();
const { currentTab } = storeToRefs(tabStore);
</script>
