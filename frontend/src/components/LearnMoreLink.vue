<template>
  <a
    v-if="url"
    class="inline-flex items-center"
    :class="[color === 'normal' ? 'normal-link' : 'light-link']"
    :href="url"
    :target="external ? '__BLANK' : ''"
  >
    {{ text }}
    <ExternalLinkIcon v-if="external" class="w-4 h-4 ml-1" />
  </a>
</template>

<script lang="ts" setup>
import { ExternalLinkIcon } from "lucide-vue-next";
import { computed } from "vue";
import { t } from "@/plugins/i18n";

const props = withDefaults(
  defineProps<{
    url: string;
    text?: string;
    color?: "normal" | "light";
  }>(),
  {
    color: "normal",
    text: () => t("common.learn-more"),
  }
);

const external = computed(() => props.url.startsWith("http"));
</script>
