<template>
  <BBModal
    :title="$t('plugin.ai.conversation.history-conversations')"
    class="!pb-0"
    container-class="!p-0"
    @close="showHistoryDialog = false"
  >
    <div
      class="h-[calc(100vh-10rem)] w-[calc(100vw-8rem)] max-w-[72rem] px-8 pb-4 flex"
    >
      <aside class="hidden lg:flex lg:flex-col w-[14em] border-l border-b">
        <ConversationList />
      </aside>

      <div class="flex-1 flex flex-col bg-gray-100">
        <ChatView mode="VIEW" :conversation="selected" />
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useAIContext, useCurrentChat } from "../../logic";
import ChatView from "../ChatView";
import ConversationList from "./ConversationList.vue";

const { showHistoryDialog } = useAIContext();
const { selected } = useCurrentChat();
</script>
