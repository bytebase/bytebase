<template>
  <div class="w-full h-full flex flex-col justify-end items-center pb-[2rem]">
    <div
      class="w-[30rem] mx-auto bg-white border border-control-border rounded shadow-md py-1"
    >
      <div class="mt-1 mb-2 px-2 text-md font-normal text-control-light">
        {{ $t("plugin.ai.examples") }}
      </div>

      <ul class="px-1 text-sm">
        <li
          v-for="(sug, i) in presetSuggestions"
          :key="i"
          class="py-2 px-1 flex items-center justify-between gap-x-1 rounded-sm hover:bg-indigo-600/10 cursor-default"
          @click="$emit('select', sug)"
        >
          <heroicons-outline:magnifying-glass class="w-4 h-4" />
          <span class="flex-1">{{ sug }}</span>
          <heroicons-outline:chevron-right class="w-4 h-4 text-gray-400" />
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

defineEmits<{
  (event: "select", preset: string): void;
}>();

const { t } = useI18n();

const presetSuggestions = computed(() => {
  return [
    t("plugin.ai.preset-suggestion.1"),
    t("plugin.ai.preset-suggestion.2"),
    t("plugin.ai.preset-suggestion.3"),
  ];
});
</script>
