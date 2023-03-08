<template>
  <div class="h-full overflow-hidden flex flex-col">
    <div class="py-2 px-2">
      <h3 class="font-bold">
        {{ $t("plugin.ai.conversation.conversations") }}
      </h3>
    </div>
    <div class="flex-1 overflow-y-auto p-1 flex flex-col gap-y-2">
      <template v-if="ready">
        <div
          v-for="conversation in conversationList"
          :key="conversation.id"
          :data-conversation-id="conversation.id"
          class="flex items-start gap-x-0.5 border rounded-md py-2 pl-2 pr-0.5 hover:bg-indigo-50 hover:border-indigo-400 cursor-pointer"
          :class="[
            selectedConversation?.id === conversation.id &&
              'bg-indigo-100 border-indigo-400',
          ]"
          @click="store.selectConversation(conversation)"
        >
          <div
            v-if="conversation.name"
            class="text-sm flex-1 whitespace-pre-wrap break-words break-all"
          >
            {{ conversation.name }}
          </div>
          <div v-else class="text-sm flex-1 truncate text-gray-500 italic">
            {{
              head(conversation.messageList)?.content ||
              $t("plugin.ai.conversation.untitled")
            }}
          </div>
          <div class="flex items-center gap-x-px">
            <button
              class="flex items-center p-0.5 border border-transparent rounded text-gray-500 hover:text-accent hover:bg-indigo-50 hover:border-accent cursor-pointer"
              @click.stop="state.rename = conversation"
            >
              <heroicons-outline:pencil class="w-3 h-3" />
            </button>
            <NPopconfirm
              @positive-click="handleDeleteConversation(conversation)"
            >
              <template #trigger>
                <button
                  class="flex items-center p-0.5 border border-transparent rounded text-gray-500 hover:text-accent hover:bg-indigo-50 hover:border-accent cursor-pointer"
                  @click.stop=""
                >
                  <heroicons-outline:trash class="w-3 h-3" />
                </button>
              </template>
              <span>{{ $t("plugin.ai.conversation.delete") }}</span>
            </NPopconfirm>
          </div>
        </div>

        <div
          class="sticky bottom-0 btn-normal items-center justify-center gap-x-1"
          @click="addConversation"
        >
          <heroicons-outline:plus class="w-4 h-4" />
          <span class="pr-3">{{ $t("common.new") }}</span>
        </div>
      </template>
      <template v-else>
        <BBSpin class="self-center mt-8" />
      </template>
    </div>

    <ConversationRenameDialog
      v-if="state.rename"
      :conversation="state.rename"
      @cancel="state.rename = undefined"
      @updated="state.rename = undefined"
    />
  </div>
</template>

<script lang="ts" setup>
import { reactive, toRef, watch } from "vue";
import { NPopconfirm } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { head } from "lodash-es";

import ConversationRenameDialog from "./ConversationRenameDialog.vue";
import type { Conversation } from "../types";
import { useConversationList, useConversationStore } from "../store";

type LocalState = {
  rename: Conversation | undefined;
};

const state = reactive<LocalState>({
  rename: undefined,
});
const store = useConversationStore();
const { conversationList, ready } = useConversationList();
const selectedConversation = toRef(store, "selectedConversation");

const addConversation = async () => {
  const c = await store.createConversation({
    name: "", // Will display as "Untitled conversation"
  });
  store.selectConversation(c);
};

const handleDeleteConversation = async (conversation: Conversation) => {
  const index = conversationList.value.findIndex(
    (c) => c.id === selectedConversation.value?.id
  );
  await store.deleteConversation(conversation.id);
  store.selectConversation(conversationList.value[index]);
};

watch(
  [() => selectedConversation.value?.id, conversationList],
  ([id, list]) => {
    if (!id) return;
    if (list.length === 0) return;
    requestAnimationFrame(() => {
      const elem = document.querySelector(`[data-conversation-id="${id}"]`);
      if (!elem) return;
      scrollIntoView(elem, { scrollMode: "if-needed" });
    });
  },
  { immediate: true }
);
</script>
