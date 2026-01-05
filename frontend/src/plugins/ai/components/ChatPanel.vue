<template>
  <div
    v-if="aiSetting.enabled"
    class="w-full h-full flex-1 flex flex-col overflow-hidden"
  >
    <ActionBar />

    <ChatView
      v-if="ready"
      :conversation="selectedConversation"
      class="pt-2"
      @enter="requestAI"
    />
    <div
      v-else
      class="flex-1 overflow-hidden relative flex items-center justify-center"
    >
      <NSpin size="small" />
    </div>

    <div class="px-2 pb-2 pt-1 flex flex-col gap-1">
      <DynamicSuggestions class="w-full" @enter="requestAI" />
      <PromptInput v-if="tab" @enter="requestAI" />
    </div>

    <HistoryPanel />
  </div>
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import { NSpin } from "naive-ui";
import { storeToRefs } from "pinia";
import { reactive, watch } from "vue";
import { sqlServiceClientConnect } from "@/connect";
import { useSQLEditorTabStore } from "@/store";
import {
  type AICompletionRequest_Message,
  AICompletionRequest_MessageSchema,
  AICompletionRequestSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { nextAnimationFrame } from "@/utils";
import { onConnectionChanged, useAIContext, useCurrentChat } from "../logic";
import * as promptUtils from "../logic/prompt";
import { useConversationStore } from "../store";
import ActionBar from "./ActionBar.vue";
import ChatView from "./ChatView";
import DynamicSuggestions from "./DynamicSuggestions.vue";
import HistoryPanel from "./HistoryPanel";
import PromptInput from "./PromptInput.vue";

type LocalState = {
  loading: boolean;
};

const state = reactive<LocalState>({
  loading: false,
});

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const store = useConversationStore();

const context = useAIContext();
const { aiSetting, showHistoryDialog, pendingSendChat } = context;
const {
  list: conversationList,
  ready,
  selected: selectedConversation,
} = useCurrentChat();

const requestAI = async (query: string) => {
  const conversation = selectedConversation.value;
  if (!conversation) return;
  const t = tab.value;
  if (!t) return;

  const { messageList } = conversation;
  if (messageList.length === 0) {
    // For the first message of a conversation,
    // add extra database schema metadata info if possible
    const engine = context.engine.value;
    const databaseMetadata = context.databaseMetadata.value;
    const schema = context.schema.value;

    const prompts: string[] = [];
    prompts.push(promptUtils.declaration(databaseMetadata, engine, schema));
    prompts.push(query);
    const prompt = prompts.join("\n");
    await store.createMessage({
      conversation_id: conversation.id,
      content: query,
      prompt,
      author: "USER",
      error: "",
      status: "DONE",
    });
    console.debug("[AI Assistant] init chat:", prompt);
  } else {
    await store.createMessage({
      conversation_id: conversation.id,
      content: query,
      prompt: query,
      author: "USER",
      error: "",
      status: "DONE",
    });
  }

  const answer = await store.createMessage({
    author: "AI",
    prompt: "",
    content: "",
    error: "",
    conversation_id: conversation.id,
    status: "LOADING",
  });
  const messages: AICompletionRequest_Message[] = [];
  conversation.messageList.forEach((message) => {
    const { author, prompt } = message;
    messages.push(
      createProto(AICompletionRequest_MessageSchema, {
        role: author === "USER" ? "user" : "assistant",
        content: prompt,
      })
    );
  });
  state.loading = true;
  try {
    const request = createProto(AICompletionRequestSchema, { messages });
    const response = await sqlServiceClientConnect.aICompletion(request);
    const text = head(head(response.candidates)?.content?.parts)?.text?.trim();
    console.debug("[AI Assistant] answer:", text);
    if (text) {
      answer.content = text;
      answer.prompt = text;
    }

    answer.status = "DONE";
  } catch (err) {
    answer.error = String(err);
    answer.status = "FAILED";
  } finally {
    state.loading = false;
    await store.updateMessage(answer);
    if (conversation.id === conversation.id) {
      if (answer.status === "FAILED") {
        context.events.emit("error", answer.error);
      }
    }
  }
};

onConnectionChanged(() => {
  showHistoryDialog.value = false;
});

watch(
  [ready, conversationList],
  async ([ready, list]) => {
    if (ready && list.length === 0) {
      store.createConversation({
        name: "",
        instance: tab.value?.connection.instance ?? "",
        database: tab.value?.connection.database ?? "",
      });
    }
  },
  { immediate: true }
);

watch(
  [ready, pendingSendChat],
  async ([ready]) => {
    if (!ready) return;

    await nextAnimationFrame();
    if (!pendingSendChat.value) return;
    requestAI(pendingSendChat.value.content);
    pendingSendChat.value = undefined;
  },
  {
    immediate: true,
    flush: "post",
  }
);
</script>
