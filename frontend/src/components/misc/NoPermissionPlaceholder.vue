<template>
  <NEmpty
    :description="description ?? $t('common.missing-required-permission')"
    v-bind="$attrs"
  >
    <template #icon>
      <ShieldAlertIcon class="w-full h-auto" stroke-width="1.25" />
    </template>
    <template #extra>
      <slot name="extra" />
    </template>
  </NEmpty>
</template>

<script lang="ts" setup>
import { ShieldAlertIcon } from "lucide-vue-next";
import { NEmpty } from "naive-ui";

defineProps({
  description: {
    type: String,
    required: false,
  },
});
</script>
