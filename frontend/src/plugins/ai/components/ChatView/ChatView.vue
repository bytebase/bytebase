<template>
  <div ref="scrollerRef" class="flex-1 py-4 overflow-y-auto">
    <template v-if="conversation">
      <PresetSuggestions
        v-if="conversation.messageList.length === 0"
        @select="$emit('enter', $event)"
      />
      <div
        v-else
        ref="containerRef"
        class="flex flex-col justify-end px-2 gap-y-8"
      >
        <div
          v-for="message in conversation?.messageList"
          :key="message.id"
          class="flex"
          :class="[message.author === 'AI' ? 'justify-start' : 'justify-end']"
        >
          <UserMessageView
            v-if="message.author === 'USER'"
            :message="message"
          />
          <AIMessageView v-if="message.author === 'AI'" :message="message" />
        </div>
      </div>
    </template>
    <template v-else>
      <div
        class="w-full h-full flex flex-col justify-end items-center pb-[2rem]"
      >
        <i18n-t
          keypath="plugin.ai.conversation.select-or-create"
          tag="p"
          class="textinfolabel"
        >
          <template #create>
            <button class="normal-link" @click="handleCreate">
              {{ $t("common.create") }}
            </button>
          </template>
        </i18n-t>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { ref, toRef, watch } from "vue";
import { useElementSize } from "@vueuse/core";

import UserMessageView from "./UserMessageView.vue";
import AIMessageView from "./AIMessageView.vue";
import PresetSuggestions from "./PresetSuggestions.vue";
import { useConversationStore } from "../../store";

defineEmits<{
  (event: "enter", value: string): void;
}>();

const scrollerRef = ref<HTMLDivElement>();
const containerRef = ref<HTMLDivElement>();

const store = useConversationStore();
const conversation = toRef(store, "selectedConversation");

const { height: containerHeight } = useElementSize(containerRef);

const scrollToLast = () => {
  const scroller = scrollerRef.value;
  const container = containerRef.value;
  if (!scroller) return;
  if (!container) return;
  scroller.scrollTo(0, container.scrollHeight);
};
watch(containerHeight, scrollToLast, { immediate: true });

const handleCreate = async () => {
  const c = await store.createConversation({
    name: "", // Will display as "Untitled conversation"
  });
  store.selectConversation(c);
};
</script>
