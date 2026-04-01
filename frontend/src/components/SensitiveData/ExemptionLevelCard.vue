<template>
  <div
    class="flex items-center gap-x-6 px-4 py-3 bg-gray-50 rounded border border-gray-200"
  >
    <div class="flex items-center gap-x-2">
      <span class="textinfolabel font-medium uppercase text-xs">{{ $t("common.scope") }}</span>
      <span
        class="px-2 py-0.5 rounded text-xs bg-green-100 text-green-700 border border-green-200"
      >
        {{ $t("database.all") }}
      </span>
    </div>
    <div class="flex items-center gap-x-2">
      <span class="textinfolabel font-medium uppercase text-xs">{{ $t("common.classification-level") }}</span>
      <LevelBadge
        v-if="classificationLevel"
        :level="classificationLevel.value"
        :operator="classificationLevel.operator"
      />
      <LevelBadge v-else :no-limit="true" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import LevelBadge from "./LevelBadge.vue";
import type { ClassificationLevel } from "./types";

defineProps<{
  classificationLevel?: ClassificationLevel;
}>();
</script>
