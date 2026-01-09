<template>
  <div class="w-full flex-1 flex flex-row items-stretch overflow-hidden">
    <Panels>
      <template #code-panel>
        <StandardPanel
          v-if="!currentTab || currentTab.mode === 'WORKSHEET'"
          :key="`standard-${currentTab?.id || 'default'}`"
        />

        <TerminalPanel
          v-else-if="currentTab.mode === 'ADMIN'"
          :key="`terminal-${currentTab?.id || 'default'}`"
        />

        <NoPermissionPlaceholder
          v-else
          :key="`no-permission-${currentTab?.id || 'default'}`"
          :description="$t('database.access-denied')"
        />
      </template>
    </Panels>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import NoPermissionPlaceholder from "@/components/Permission/NoPermissionPlaceholder.vue";
import { useSQLEditorTabStore } from "@/store";
import Panels from "./Panels";
import StandardPanel from "./StandardPanel";
import TerminalPanel from "./TerminalPanel";

const tabStore = useSQLEditorTabStore();
const { currentTab } = storeToRefs(tabStore);
</script>
