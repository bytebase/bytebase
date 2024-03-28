<template>
  <div class="shrink-0 w-6 h-6 flex items-center justify-center" @click.stop>
    <NTooltip>
      <template #trigger>
        <carbon:dot-mark class="text-accent w-4 h-4" />
      </template>
      <template #default>
        <span>{{ $t("sql-editor.tab.unsaved") }}</span>
      </template>
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { NTooltip } from "naive-ui";
</script>
