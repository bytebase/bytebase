<template>
  <div
    class="w-full flex flex-wrap gap-y-1 justify-between sm:items-center px-1 py-1 border-b bg-white"
  >
    <div
      class="action-left gap-x-2 flex overflow-x-auto sm:overflow-x-hidden items-center pl-1"
    >
      <h3>{{ $t("plugin.ai.ai-assistant") }}</h3>
    </div>

    <div
      class="action-right gap-x-1 flex overflow-x-auto sm:overflow-x-hidden sm:justify-end items-center"
    >
      <NPopover placement="bottom">
        <template #trigger>
          <NButton
            size="small"
            quaternary
            style="--n-padding: 0 5px"
            @click="showHistoryDialog = true"
          >
            <template #icon>
              <ClockIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          {{ $t("plugin.ai.conversation.view-history-conversations") }}
        </template>
      </NPopover>
      <NPopover placement="bottom">
        <template #trigger>
          <NButton
            size="small"
            quaternary
            style="--n-padding: 0 5px"
            @click="events.emit('new-conversation', { input: '' })"
          >
            <template #icon>
              <PlusIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          {{ $t("plugin.ai.conversation.new-conversation") }}
        </template>
      </NPopover>
      <NPopover placement="bottom">
        <template #trigger>
          <NButton
            size="small"
            quaternary
            style="--n-padding: 0 5px"
            @click="showAIPanel = false"
          >
            <template #icon>
              <XIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          {{ $t("common.hide") }}
        </template>
      </NPopover>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ClockIcon, XIcon, PlusIcon } from "lucide-vue-next";
import { NButton, NPopover } from "naive-ui";
import { useSQLEditorContext } from "@/views/sql-editor/context";
import { useAIContext } from "../logic";

const { events, showHistoryDialog } = useAIContext();
const { showAIPanel } = useSQLEditorContext();
</script>
