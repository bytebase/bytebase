<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <HeroiconsOutlineLightningBolt
        class="w-4 h-auto outline-none"
        @click="handleTriggerClick"
      />
    </template>
    {{ $t("sql-editor.open-connection") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import HeroiconsOutlineLightningBolt from "~icons/heroicons-outline/lightning-bolt.vue";

const emit = defineEmits(["click"]);

const handleTriggerClick = (e: MouseEvent) => {
  e.preventDefault();
  e.stopPropagation();
  emit("click");
};
</script>
