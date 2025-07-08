<template>
  <a
    class="inline-flex items-center"
    :class="[color === 'normal' ? 'normal-link' : 'light-link']"
    :href="url"
    :target="external ? '__BLANK' : ''"
  >
    {{ $t("common.learn-more") }}
    <ExternalLinkIcon v-if="external" class="w-4 h-4 ml-1" />
  </a>
</template>

<script lang="ts" setup>
import { ExternalLinkIcon } from "lucide-vue-next";

withDefaults(
  defineProps<{
    url: string;
    external?: boolean;
    color?: "normal" | "light";
  }>(),
  {
    color: "normal",
    external: true,
  }
);
</script>
