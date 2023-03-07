<template>
  <BBModal class="!pb-0" container-class="!p-0" @close="$emit('close')">
    <div
      class="h-[calc(100vh-10rem)] w-[calc(100vw-8rem)] max-w-[72rem] px-8 pb-4 flex"
    >
      <aside class="hidden lg:flex lg:flex-col w-[14em] border-l border-b">
        <ConversationList />
      </aside>

      <div class="flex-1 flex flex-col bg-gray-100">
        <ChatView @enter="requestAI" />

        <div class="px-2 py-2">
          <div>
            <label class="inline-flex items-center gap-x-1">
              <BBCheckbox :value="autoRun" @toggle="autoRun = $event" />
              <span class="textinfolabel">
                {{ $t("plugin.ai.run-automatically") }}
              </span>
            </label>
          </div>
          <PromptInput :disabled="state.loading" @enter="requestAI" />
        </div>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive, toRef } from "vue";
import { Axios, AxiosResponse } from "axios";
import { head } from "lodash-es";

import { BBCheckbox, BBModal } from "@/bbkit";
import { useAIContext } from "../logic";
import type { OpenAIMessage, OpenAIResponse } from "../types";
import ConversationList from "./ConversationList.vue";
import ChatView from "./ChatView";
import PromptInput from "./PromptInput.vue";
import { useConversationStore } from "../store";

type LocalState = {
  loading: boolean;
};

defineEmits<{
  (event: "close"): void;
}>();

const state = reactive<LocalState>({
  loading: false,
});

const context = useAIContext();

const { showDialog, openAIKey, autoRun } = context;

const store = useConversationStore();
const conversation = toRef(store, "selectedConversation");

const requestAI = async (query: string) => {
  const c = conversation.value;
  if (!c) return;

  const { messageList } = c;
  if (messageList.length === 0) {
    // For the first message of a conversation,
    // add extra database schema metadata info if possible
    const engineType = context.engineType.value;
    const databaseMetadata = context.databaseMetadata.value;
    const prompts: string[] = [];
    if (engineType) {
      if (databaseMetadata) {
        prompts.push(`### ${engineType} tables, with their properties:`);
      } else {
        prompts.push(`### ${engineType} database`);
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
      conversation_id: c.id,
      content: query,
      prompt,
      author: "USER",
      error: "",
      status: "DONE",
    });
  } else {
    await store.createMessage({
      conversation_id: c.id,
      content: query,
      prompt: query,
      author: "USER",
      error: "",
      status: "DONE",
    });
  }

  const answer = await store.createMessage({
    author: "AI",
    prompt: "SELECT",
    content: "",
    error: "",
    conversation_id: c.id,
    status: "LOADING",
  });
  const url = "https://api.openai.com/v1/chat/completions";
  const FINAL_PROMPT = "SELECT";
  const messages: OpenAIMessage[] = [];

  c.messageList.forEach((message) => {
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
      JSON.stringify(body, null, "  "),
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
      const parts = [text];
      if (!text.startsWith(FINAL_PROMPT)) {
        parts.unshift(FINAL_PROMPT);
      }
      const statement = parts.join(" ").trim();

      answer.content = statement;
      answer.prompt = statement;
    }

    answer.status = "DONE";
  } catch (err) {
    answer.error = String(err);
    answer.status = "FAILED";
  } finally {
    state.loading = false;
    await store.updateMessage(answer);
    if (c.id === conversation.value?.id) {
      if (answer.status === "FAILED") {
        context.events.emit("error", answer.error);
      } else {
        if (showDialog.value && autoRun.value) {
          context.events.emit("apply-statement", {
            statement: answer.content,
            run: autoRun.value,
          });
        }
      }
    }
  }
};
</script>
