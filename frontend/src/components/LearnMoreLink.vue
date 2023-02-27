<template>
  <a
    class="inline-flex items-center"
    :class="[color === 'normal' ? 'normal-link' : 'light-link']"
    :href="url"
    target="__BLANK"
  >
    {{ $t("common.learn-more") }}
    <heroicons-outline:external-link class="w-4 h-4 ml-1" />
  </a>
</template>

<script lang="ts" setup>
import { PropType } from "vue";

defineProps({
  url: {
    type: String,
    required: true,
  },
  color: {
    type: String as PropType<"normal" | "light">,
    default: "normal",
  },
});
</script>
