<template>
  <a
    v-if="!hide"
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
import { computed } from "vue";
import { useActuatorV1Store } from "@/store";

const props = withDefaults(
  defineProps<{
    url: string;
    external?: boolean;
    color?: "normal" | "light";
    hideWhenEmbedded?: boolean;
  }>(),
  {
    color: "normal",
    hideWhenEmbedded: false,
    external: true,
  }
);

const hide = computed(() => {
  if (!props.hideWhenEmbedded) return false;
  return useActuatorV1Store().appProfile.embedded;
});
</script>
