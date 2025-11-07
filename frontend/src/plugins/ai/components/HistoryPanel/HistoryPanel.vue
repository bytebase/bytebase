<template>
  <Drawer v-model:show="showHistoryDialog">
    <DrawerContent
      :title="$t('plugin.ai.conversation.history-conversations')"
      class="w-[calc(100vw-8rem)] max-w-6xl"
      body-content-style="padding: 0"
    >
      <div class="flex h-full">
        <aside class="hidden lg:flex lg:flex-col w-[14em] border-l border-b">
          <ConversationList />
        </aside>

        <div class="flex-1 flex flex-col py-2 bg-gray-100 overflow-hidden">
          <ChatView mode="VIEW" :conversation="selected" />
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { Drawer, DrawerContent } from "@/components/v2";
import { useAIContext, useCurrentChat } from "../../logic";
import ChatView from "../ChatView";
import ConversationList from "./ConversationList.vue";

const { showHistoryDialog } = useAIContext();
const { selected } = useCurrentChat();
</script>
