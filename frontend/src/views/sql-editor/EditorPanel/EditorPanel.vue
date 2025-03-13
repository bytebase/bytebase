<template>
  <div class="w-full flex-1 flex flex-row items-stretch overflow-hidden">
    <Panels>
      <template #code-panel>
        <StandardPanel v-if="!currentTab || currentTab.mode === 'WORKSHEET'" />

        <TerminalPanel v-else-if="currentTab.mode === 'ADMIN'" />

        <NoPermissionPlaceholder
          v-else
          :description="$t('database.access-denied')"
        />
      </template>
    </Panels>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from "pinia";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useSQLEditorTabStore } from "@/store";
import Panels from "./Panels";
import StandardPanel from "./StandardPanel";
import TerminalPanel from "./TerminalPanel";

const tabStore = useSQLEditorTabStore();
const { currentTab } = storeToRefs(tabStore);
</script>
