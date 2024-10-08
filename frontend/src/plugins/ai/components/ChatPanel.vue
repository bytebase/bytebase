<template>
  <div
    v-if="openAIKey"
    class="w-full h-full flex-1 flex flex-col overflow-hidden"
  >
    <ActionBar />

    <ChatView :conversation="selectedConversation" @enter="requestAI" />

    <div class="px-2 pb-2 flex flex-col gap-2">
      <div class="flex items-center gap-2 w-full">
        <DynamicSuggestions class="flex-1" @enter="requestAI" />
      </div>
      <PromptInput v-if="tab" @enter="requestAI" />
    </div>

    <HistoryPanel />
  </div>
</template>

<script lang="ts" setup>
import type { AxiosResponse } from "axios";
import { Axios } from "axios";
import { head } from "lodash-es";
import { storeToRefs } from "pinia";
import { reactive, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore } from "@/store";
import { onConnectionChanged, useAIContext, useCurrentChat } from "../logic";
import * as promptUtils from "../logic/prompt";
import { useConversationStore } from "../store";
import type { OpenAIMessage, OpenAIResponse } from "../types";
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
const { events, openAIKey, openAIEndpoint, showHistoryDialog } = context;
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

    const prompts: string[] = [];
    prompts.push(promptUtils.declaration(databaseMetadata, engine));
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
    console.log(prompt);
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
  const url =
    openAIEndpoint.value === ""
      ? "https://api.openai.com/v1/chat/completions"
      : openAIEndpoint.value + "/v1/chat/completions";
  const messages: OpenAIMessage[] = [];

  conversation.messageList.forEach((message) => {
    const { author, prompt } = message;
    messages.push({
      role: author === "USER" ? "user" : "assistant",
      content: prompt,
    });
  });
  const body = {
    model: "gpt-3.5-turbo",
    messages,
    temperature: 0,
    stop: ["#", ";"],
    top_p: 1.0,
    frequency_penalty: 0.0,
    presence_penalty: 0.0,
  };
  state.loading = true;
  const axios = new Axios({
    timeout: 300 * 1000,
    responseType: "json",
  });
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${openAIKey.value}`,
  };
  try {
    const response: AxiosResponse<string> = await axios.post(
      url,
      JSON.stringify(body),
      {
        headers,
      }
    );

    const data = JSON.parse(response.data) as OpenAIResponse;
    if (data?.error) {
      throw new Error(data.error.message);
    }

    const text = head(data?.choices)?.message.content?.trim();
    if (text) {
      answer.content = text;
      answer.prompt = text;
      console.log(answer.content);
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

useEmitteryEventListener(events, "new-conversation", async () => {
  if (!tab.value) return;
  showHistoryDialog.value = false;
  const c = await store.createConversation({
    name: "",
    ...tab.value.connection,
  });
  selectedConversation.value = c;
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
</script>
