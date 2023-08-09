<template>
  <div
    v-if="openAIKey"
    class="w-full flex flex-col"
    :class="[
      !isChatMode && 'px-4 py-2 border-t',
      isChatMode && 'flex-1 overflow-hidden',
    ]"
  >
    <template v-if="isChatMode">
      <ActionBar />
      <ChatView :conversation="selectedConversation" @enter="requestAI" />
    </template>

    <div :class="[isChatMode && 'px-4 py-2 flex flex-col gap-2']">
      <div v-if="isChatMode" class="flex items-center gap-2 w-full">
        <DynamicSuggestions class="flex-1" @enter="requestAI" />
      </div>
      <PromptInput @focus="tab.editMode = 'CHAT-TO-SQL'" @enter="requestAI" />
    </div>

    <template v-if="isChatMode">
      <HistoryPanel v-if="showHistoryDialog" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { Axios, AxiosResponse } from "axios";
import { head } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { useCurrentTab } from "@/store";
import { engineNameV1 } from "@/utils";
import { onConnectionChanged, useAIContext, useCurrentChat } from "../logic";
import { useConversationStore } from "../store";
import { OpenAIMessage, OpenAIResponse } from "../types";
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

const tab = useCurrentTab();
const store = useConversationStore();
const isChatMode = computed(() => tab.value.editMode === "CHAT-TO-SQL");

const context = useAIContext();
const { events, openAIKey, openAIEndpoint, autoRun, showHistoryDialog } =
  context;
const {
  list: conversationList,
  ready,
  selected: selectedConversation,
} = useCurrentChat();

const requestAI = async (query: string) => {
  const conversation = selectedConversation.value;
  if (!conversation) return;
  const t = tab.value;

  const { messageList } = conversation;
  if (messageList.length === 0) {
    // For the first message of a conversation,
    // add extra database schema metadata info if possible
    const engine = context.engine.value;
    const databaseMetadata = context.databaseMetadata.value;
    const prompts: string[] = [];
    if (engine) {
      if (databaseMetadata) {
        prompts.push(
          `### ${engineNameV1(engine)} tables, with their properties:`
        );
      } else {
        prompts.push(`### ${engineNameV1(engine)} database`);
      }
    } else {
      if (databaseMetadata) {
        prompts.push(`### Giving a database`);
      }
    }
    if (databaseMetadata) {
      databaseMetadata.schemas.forEach((schema) => {
        schema.tables.forEach((table) => {
          const name = schema.name
            ? `${schema.name}.${table.name}`
            : table.name;
          const columns = table.columns.map((column) => column.name).join(", ");
          prompts.push(`# ${name}(${columns})`);
        });
      });
    }
    prompts.push(`### Write a SQL statement to solve the question below`);
    prompts.push(`### ${query}`);
    const prompt = prompts.join("\n");
    await store.createMessage({
      conversation_id: conversation.id,
      content: query,
      prompt,
      author: "USER",
      error: "",
      status: "DONE",
    });
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
    timeout: 20 * 1000,
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
      } else {
        if (
          autoRun.value &&
          t.id === tab.value.id &&
          conversation.id === selectedConversation.value?.id
        ) {
          // If the chat is still active, emit 'apply-statement' event
          context.events.emit("apply-statement", {
            statement: answer.content,
            run: autoRun.value,
          });
        }
      }
    }
  }
};

onConnectionChanged(() => {
  showHistoryDialog.value = false;
});

events.on("new-conversation", async () => {
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
        ...tab.value.connection,
      });
    }
  },
  { immediate: true }
);
</script>
