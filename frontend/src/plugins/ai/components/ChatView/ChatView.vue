<template>
  <div ref="scrollerRef" class="flex-1 py-4 overflow-y-auto">
    <template v-if="conversation">
      <template v-if="conversation.messageList.length === 0">
        <EmptyView v-if="mode === 'VIEW'" />
      </template>
      <div
        v-else
        ref="containerRef"
        class="flex flex-col justify-end px-4 gap-y-8"
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
    <template v-else-if="mode === 'CHAT'">
      <div
        class="w-full h-full flex flex-col justify-end items-center pb-[2rem]"
      >
        <i18n-t
          keypath="plugin.ai.conversation.select-or-create"
          tag="p"
          class="textinfolabel"
        >
          <template #create>
            <button
              class="normal-link"
              @click="events.emit('new-conversation')"
            >
              {{ $t("common.create") }}
            </button>
          </template>
        </i18n-t>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { ref, toRef, watch } from "vue";
import { useAIContext } from "../../logic";
import type { Conversation } from "../../types";
import AIMessageView from "./AIMessageView.vue";
import EmptyView from "./EmptyView.vue";
import UserMessageView from "./UserMessageView.vue";
import { provideChatViewContext } from "./context";
import type { Mode } from "./types";

const props = withDefaults(
  defineProps<{
    mode?: Mode;
    conversation?: Conversation;
  }>(),
  {
    mode: "CHAT",
    conversation: undefined,
  }
);

defineEmits<{
  (event: "enter", value: string): void;
}>();

const scrollerRef = ref<HTMLDivElement>();
const containerRef = ref<HTMLDivElement>();

const { events } = useAIContext();

const { height: containerHeight } = useElementSize(containerRef);

const scrollToLast = () => {
  const scroller = scrollerRef.value;
  const container = containerRef.value;
  if (!scroller) return;
  if (!container) return;
  scroller.scrollTo(0, container.scrollHeight);
};
watch(containerHeight, scrollToLast, { immediate: true });

provideChatViewContext({
  mode: toRef(props, "mode"),
});
</script>
