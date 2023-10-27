<template>
  <a
    class="inline-flex items-center"
    :class="[color === 'normal' ? 'normal-link' : 'light-link']"
    :href="url"
    target="__BLANK"
  >
    🐞 {{ $t("common.troubleshoot") }}
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
