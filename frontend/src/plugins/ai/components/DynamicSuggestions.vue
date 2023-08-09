<template>
  <div class="flex items-center overflow-hidden h-[22px]">
    <template v-if="dynamicSuggestions && !ready">
      <BBSpin class="w-4 h-4 mr-2" />
      <span class="text-sm">{{
        $t("plugin.ai.conversation.tips.suggest-prompt")
      }}</span>
    </template>

    <template v-if="ready && suggestions.length > 0">
      <div
        class="relative flex items-center gap-x-2 overflow-hidden text-xs leading-4"
      >
        <div
          class="flex items-stretch gap-x-2 whitespace-nowrap overflow-x-auto hide-scrollbar"
        >
          <NEllipsis
            v-for="(sug, i) in suggestions"
            :key="i"
            style="max-width: 20rem"
            class="border shrink-0 py-0.5 px-2 cursor-pointer hover:bg-indigo-100 hover:border-indigo-500"
            @click.capture="consume(sug)"
          >
            {{ sug }}
          </NEllipsis>

          <div class="shrink-0 flex items-center">
            <BBSpin v-if="state === 'LOADING'" class="w-4 h-4" />
            <button
              v-if="state === 'IDLE'"
              class="text-gray-500 cursor-pointer py-0.5 px-1 rounded-md hover:bg-gray-200"
              @click="dynamicSuggestions?.fetch()"
            >
              {{ $t("plugin.ai.conversation.tips.more") }}
            </button>
            <span v-if="state === 'ENDED'" class="text-gray-500">
              {{ $t("plugin.ai.conversation.tips.no-more") }}
            </span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed } from "vue";
import { BBSpin } from "@/bbkit";
import { useDynamicSuggestions } from "../logic";

const emit = defineEmits<{
  (event: "enter", query: string): void;
}>();

const dynamicSuggestions = useDynamicSuggestions();

const ready = computed(() => dynamicSuggestions.value?.ready ?? false);
const state = computed(() => dynamicSuggestions.value?.state ?? "IDLE");
const suggestions = computed(() => dynamicSuggestions.value?.suggestions ?? []);

const consume = (sug: string) => {
  const suggestions = dynamicSuggestions.value;
  if (!suggestions) return;
  suggestions.consume(sug);
  emit("enter", sug);
};
</script>
