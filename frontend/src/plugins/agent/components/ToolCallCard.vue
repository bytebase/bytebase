<script setup lang="ts">
import { computed, ref } from "vue";
import type {
  AgentAskUserOption,
  AgentAskUserResponse,
  ToolCall,
} from "../logic/types";

const props = defineProps<{
  toolCall: ToolCall;
  result?: string;
}>();

const expanded = ref(false);

const parseJson = (value?: string): unknown => {
  if (!value) {
    return undefined;
  }
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
};

const formatJson = (value?: string): string => {
  if (!value) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
};

const parseAskUserOption = (value: unknown): AgentAskUserOption | null => {
  if (!value || typeof value !== "object") {
    return null;
  }

  const option = value as Record<string, unknown>;
  const label =
    typeof option.label === "string"
      ? option.label.trim()
      : typeof option.value === "string"
        ? option.value.trim()
        : "";
  const optionValue =
    typeof option.value === "string" ? option.value.trim() : "";

  if (!label || !optionValue) {
    return null;
  }

  return {
    label,
    value: optionValue,
    description:
      typeof option.description === "string" && option.description.trim()
        ? option.description.trim()
        : undefined,
  };
};

const resultText = computed(() => props.result ?? "");
const parsedArguments = computed(() => parseJson(props.toolCall.arguments));
const parsedResult = computed(() => parseJson(resultText.value));
const isAskUser = computed(() => props.toolCall.name === "ask_user");
const isDone = computed(() => props.toolCall.name === "done");
const askPrompt = computed(() => {
  if (
    typeof parsedArguments.value === "object" &&
    parsedArguments.value &&
    "prompt" in parsedArguments.value &&
    typeof parsedArguments.value.prompt === "string"
  ) {
    return parsedArguments.value.prompt;
  }
  return "";
});
const askKind = computed(() => {
  if (
    typeof parsedArguments.value === "object" &&
    parsedArguments.value &&
    "kind" in parsedArguments.value
  ) {
    if (parsedArguments.value.kind === "confirm") {
      return "confirm";
    }
    if (parsedArguments.value.kind === "choose") {
      return "choose";
    }
  }
  return "input";
});
const askDefaultValue = computed(() => {
  if (
    typeof parsedArguments.value === "object" &&
    parsedArguments.value &&
    "defaultValue" in parsedArguments.value &&
    typeof parsedArguments.value.defaultValue === "string"
  ) {
    return parsedArguments.value.defaultValue;
  }
  return "";
});
const askOptions = computed<AgentAskUserOption[]>(() => {
  if (
    typeof parsedArguments.value !== "object" ||
    !parsedArguments.value ||
    !("options" in parsedArguments.value) ||
    !Array.isArray(parsedArguments.value.options)
  ) {
    return [];
  }

  return parsedArguments.value.options
    .map((option) => parseAskUserOption(option))
    .filter((option): option is AgentAskUserOption => !!option);
});
const askResponse = computed<AgentAskUserResponse | null>(() => {
  if (
    typeof parsedResult.value !== "object" ||
    !parsedResult.value ||
    !("kind" in parsedResult.value) ||
    typeof parsedResult.value.kind !== "string" ||
    !("answer" in parsedResult.value) ||
    typeof parsedResult.value.answer !== "string"
  ) {
    return null;
  }

  const response = parsedResult.value as Record<string, unknown>;

  if (response.kind === "confirm") {
    if (typeof response.confirmed !== "boolean") {
      return null;
    }
    return response as AgentAskUserResponse;
  }

  if (response.kind === "choose") {
    if (typeof response.value !== "string") {
      return null;
    }
    return response as AgentAskUserResponse;
  }

  if (response.kind === "input") {
    return response as AgentAskUserResponse;
  }

  return null;
});
const doneText = computed(() => {
  if (
    typeof parsedArguments.value === "object" &&
    parsedArguments.value &&
    "text" in parsedArguments.value &&
    typeof parsedArguments.value.text === "string"
  ) {
    return parsedArguments.value.text;
  }
  return "";
});
const doneSuccess = computed(() => {
  if (
    typeof parsedArguments.value === "object" &&
    parsedArguments.value &&
    "success" in parsedArguments.value
  ) {
    return parsedArguments.value.success !== false;
  }
  return true;
});
</script>

<template>
  <div class="rounded border bg-gray-50 text-xs">
    <div
      class="flex cursor-pointer items-center gap-x-2 px-2 py-1.5"
      @click="expanded = !expanded"
    >
      <span class="font-mono text-gray-600">{{ toolCall.name }}</span>
      <template v-if="isAskUser">
        <span v-if="resultText" class="text-green-500">
          {{ $t("agent.tool-response-submitted") }}
        </span>
        <span v-else class="text-amber-600">{{ $t("agent.tool-ask-user") }}</span>
      </template>
      <template v-else-if="isDone">
        <span :class="doneSuccess ? 'text-green-500' : 'text-red-500'">
          {{ doneSuccess ? $t("agent.tool-completed") : $t("agent.tool-failed") }}
        </span>
      </template>
      <template v-else>
        <span v-if="resultText" class="text-green-500">&#10003;</span>
        <span v-else class="animate-pulse text-gray-400">&#9679;</span>
      </template>
      <span class="ml-auto text-gray-400">{{ expanded ? "\u25BE" : "\u25B8" }}</span>
    </div>

    <div v-if="expanded" class="space-y-1 border-t px-2 py-1.5">
      <template v-if="isAskUser">
        <div class="text-gray-500">
          {{
            askKind === "confirm"
              ? $t("agent.tool-ask-user-confirm")
              : askKind === "choose"
                ? $t("agent.tool-ask-user-choose")
                : $t("agent.tool-ask-user-input")
          }}
        </div>
        <div class="text-gray-500">{{ $t("agent.tool-prompt") }}</div>
        <pre class="whitespace-pre-wrap break-all text-gray-700">{{ askPrompt }}</pre>
        <template v-if="askDefaultValue">
          <div class="text-gray-500">{{ $t("agent.tool-default-value") }}</div>
          <pre class="whitespace-pre-wrap break-all text-gray-700">{{
            askDefaultValue
          }}</pre>
        </template>
        <template v-if="askKind === 'choose' && askOptions.length > 0">
          <div class="text-gray-500">{{ $t("agent.tool-options") }}</div>
          <div class="space-y-1">
            <div
              v-for="option in askOptions"
              :key="option.value"
              class="rounded border border-gray-200 bg-white px-2 py-1"
            >
              <div class="font-medium text-gray-700">{{ option.label }}</div>
              <div class="text-gray-500">{{ option.value }}</div>
              <div v-if="option.description" class="text-gray-500">
                {{ option.description }}
              </div>
            </div>
          </div>
        </template>
        <template v-if="askResponse">
          <div class="text-gray-500">{{ $t("agent.tool-answer") }}</div>
          <pre class="whitespace-pre-wrap break-all text-gray-700">{{
            askResponse.answer
          }}</pre>
          <template v-if="askResponse.kind === 'choose'">
            <div class="text-gray-500">{{ $t("agent.tool-value") }}</div>
            <pre class="whitespace-pre-wrap break-all text-gray-700">{{
              askResponse.value
            }}</pre>
          </template>
        </template>
      </template>

      <template v-else-if="isDone">
        <div class="text-gray-500">{{ $t("agent.result") }}</div>
        <pre class="whitespace-pre-wrap break-all text-gray-700">{{ doneText }}</pre>
      </template>

      <template v-else>
        <div class="text-gray-500">{{ $t("agent.args") }}</div>
        <pre class="whitespace-pre-wrap break-all text-gray-700">{{
          formatJson(toolCall.arguments)
        }}</pre>
        <template v-if="resultText">
          <div class="text-gray-500">{{ $t("agent.result") }}</div>
          <pre
            class="max-h-32 overflow-y-auto whitespace-pre-wrap break-all text-gray-700"
            >{{ formatJson(resultText) }}</pre
          >
        </template>
      </template>
    </div>
  </div>
</template>
