<template>
  <div class="h-full overflow-hidden flex flex-col">
    <div class="flex-1 overflow-y-auto p-1 flex flex-col gap-y-2">
      <template v-if="ready">
        <div
          v-for="conversation in list"
          :key="conversation.id"
          :data-conversation-id="conversation.id"
          class="flex items-start gap-x-0.5 border rounded-md py-2 pl-2 pr-0.5 hover:bg-indigo-50 hover:border-indigo-400 cursor-pointer"
          :class="[
            selected?.id === conversation.id &&
              'bg-indigo-100 border-indigo-400',
          ]"
          @click="selected = conversation"
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
          @click="events.emit('new-conversation')"
        >
          <heroicons-outline:plus class="w-4 h-4" />
          <span class="pr-2">{{
            $t("plugin.ai.conversation.new-conversation")
          }}</span>
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
import { head } from "lodash-es";
import { NPopconfirm } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { reactive, watch } from "vue";
import { useCurrentTab } from "@/store";
import { useAIContext, useCurrentChat } from "../../logic";
import { useConversationStore } from "../../store";
import type { Conversation } from "../../types";
import ConversationRenameDialog from "./ConversationRenameDialog.vue";

type LocalState = {
  rename: Conversation | undefined;
};

const tab = useCurrentTab();
const store = useConversationStore();
const { events } = useAIContext();
const { list, ready, selected } = useCurrentChat();

const state = reactive<LocalState>({
  rename: undefined,
});

watch(
  [
    () => tab.value.connection.instanceId,
    () => tab.value.connection.databaseId,
  ],
  () => {
    state.rename = undefined;
  },
  { immediate: true }
);

const handleDeleteConversation = async (conversation: Conversation) => {
  // try to keep the selected or a nearby item selected.
  const index = list.value.findIndex((c) => c.id === selected.value?.id);
  await store.deleteConversation(conversation.id);
  selected.value = list.value[index];
};

watch(
  [() => selected.value?.id, list],
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
