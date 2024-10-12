<template>
  <div v-if="show" class="flex items-center overflow-hidden h-[22px]">
    <template v-if="dynamicSuggestions && !ready">
      <BBSpin :size="16" class="mr-2" />
      <span class="text-sm">
        {{ $t("plugin.ai.conversation.tips.suggest-prompt") }}
      </span>
    </template>

    <div
      v-if="ready"
      class="relative flex items-center gap-1 overflow-hidden text-xs leading-4"
    >
      <NButton
        v-if="current"
        size="tiny"
        class="flex-1 overflow-hidden"
        @click.capture="consume"
      >
        <div class="w-full truncate">
          {{ current }}
        </div>
      </NButton>

      <BBSpin v-if="state === 'LOADING'" :size="16" class="shrink-0" />
      <NButton
        v-if="state === 'IDLE'"
        size="tiny"
        type="primary"
        quaternary
        class="shrink-0"
        @click="dynamicSuggestions?.consume()"
      >
        <RefreshCwIcon class="w-3.5 h-3.5" />
      </NButton>
      <span v-if="state === 'ENDED'" class="shrink-0 text-gray-500">
        {{ $t("plugin.ai.conversation.tips.no-more") }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { RefreshCwIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
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
const current = computed(() => dynamicSuggestions.value?.current());

const show = computed(() => {
  if (!ready.value) return true; // show a spinner
  return suggestions.value.length > 0 || state.value === "LOADING";
});

const consume = () => {
  const suggestion = dynamicSuggestions.value;
  if (!suggestion) return;
  const curr = current.value;
  if (!curr) return;
  emit("enter", curr);
  suggestion.consume();
};
</script>
