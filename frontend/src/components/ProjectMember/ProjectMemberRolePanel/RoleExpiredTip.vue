<template>
  <NTooltip>
    <template #trigger>
      <div class="flex items-center gap-x-2 line-through">
        <heroicons-outline:information-circle class="w-4 h-4 text-yellow-600" />
        {{ content }}
      </div>
    </template>
    {{ $t("role.expired-tip") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";

defineProps<{
  content: string;
}>();
</script>
