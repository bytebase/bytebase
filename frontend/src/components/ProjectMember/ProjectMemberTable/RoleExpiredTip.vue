<template>
  <NTooltip>
    <template #trigger>
      <heroicons-outline:information-circle class="w-4 h-4 text-yellow-600" />
    </template>
    {{ $t("role.expired-tip") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
</script>
