<template>
  <div
    v-if="!ready || suggestions.length > 0"
    class="flex items-center overflow-hidden h-[22px]"
  >
    <template v-if="dynamicSuggestions && !ready">
      <BBSpin :size="20" class="mr-2" />
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
            class="border rounded-sm shrink-0 py-0.5 px-2 cursor-pointer hover:bg-indigo-100 hover:border-indigo-500"
            @click.capture="consume(sug)"
          >
            {{ sug }}
          </NEllipsis>

          <div class="shrink-0 flex items-center">
            <BBSpin v-if="state === 'LOADING'" :size="16" />
            <NButton
              v-if="state === 'IDLE'"
              size="tiny"
              quaternary
              @click="dynamicSuggestions?.fetch()"
            >
              {{ $t("plugin.ai.conversation.tips.more") }}
            </NButton>
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
import { NButton, NEllipsis } from "naive-ui";
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
