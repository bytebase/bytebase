<template>
  <div
    class="w-full flex flex-wrap gap-y-2 justify-between sm:items-center px-4 py-2 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center"
    >
      <NButton size="small" @click="tab.editMode = 'SQL-EDITOR'">
        <template #icon>
          <heroicons:chevron-left />
        </template>
        <span>{{ $t("plugin.ai.conversation.exit-conversation-mode") }}</span>
      </NButton>
    </div>

    <div
      class="action-right gap-x-2 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center"
    >
      <NButton size="small" @click="showHistoryDialog = true">
        <template #icon>
          <heroicons:clock />
        </template>
        <span>{{
          $t("plugin.ai.conversation.view-history-conversations")
        }}</span>
      </NButton>
      <NButton size="small" @click="events.emit('new-conversation')">
        <template #icon>
          <heroicons:plus />
        </template>
        <span>{{ $t("plugin.ai.conversation.new-conversation") }}</span>
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useCurrentTab } from "@/store";
import { useAIContext } from "../logic";

const tab = useCurrentTab();

const { events, showHistoryDialog } = useAIContext();
</script>
