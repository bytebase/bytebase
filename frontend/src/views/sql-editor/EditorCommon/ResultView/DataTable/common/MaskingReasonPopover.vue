<template>
  <NPopover trigger="hover" :delay="100">
    <template #trigger>
      <div class="inline-flex items-center gap-0.5">
        <!-- Show semantic type icon if available -->
        <img
          v-if="props.reason.semanticTypeIcon"
          :src="props.reason.semanticTypeIcon"
          class="w-3 h-3 object-contain"
          alt=""
        />
        <heroicons:eye-slash
          class="w-3 h-3 cursor-pointer text-gray-500 hover:text-gray-700"
          @click="handleClick"
        />
      </div>
    </template>
    <div class="flex flex-col gap-y-2 max-w-sm">
      <div class="font-medium flex items-center gap-2">
        <img
          v-if="props.reason.semanticTypeIcon"
          :src="props.reason.semanticTypeIcon"
          class="w-4 h-4 object-contain"
          alt=""
        />
        {{ $t("masking.reason.title") }}
      </div>

      <div v-if="props.reason.semanticTypeTitle" class="text-sm">
        <span class="text-gray-500"
          >{{ $t("masking.reason.semantic-type") }}:</span
        >
        <span class="ml-1">{{ props.reason.semanticTypeTitle }}</span>
      </div>

      <div v-if="props.reason.algorithm" class="text-sm">
        <span class="text-gray-500">{{ $t("masking.reason.algorithm") }}:</span>
        <span class="ml-1">{{ formatAlgorithm(props.reason.algorithm) }}</span>
      </div>

      <div v-if="props.reason.context" class="text-sm">
        <span class="text-gray-500">{{ $t("masking.reason.context") }}:</span>
        <span class="ml-1">{{ props.reason.context }}</span>
      </div>

      <div v-if="props.reason.classificationLevel" class="text-sm">
        <span class="text-gray-500"
          >{{ $t("masking.reason.classification") }}:</span
        >
        <span class="ml-1">{{ props.reason.classificationLevel }}</span>
      </div>
    </div>
  </NPopover>
</template>

<script setup lang="ts">
import { NPopover } from "naive-ui";
import { useI18n } from "vue-i18n";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";

const props = defineProps<{
  reason: MaskingReason;
}>();

const emit = defineEmits<{
  (event: "click"): void;
}>();

const { t } = useI18n();

const formatAlgorithm = (algorithm: string): string => {
  const algorithmKey = algorithm.toLowerCase().replace(/\s+/g, "-");
  const key = `masking.algorithms.${algorithmKey}`;
  // Check if translation exists
  const translated = t(key);
  if (translated === key) {
    // If no translation found, return original
    return algorithm;
  }
  return translated;
};

const handleClick = () => {
  emit("click");
};
</script>
