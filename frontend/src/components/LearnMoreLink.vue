<template>
  <a
    v-if="!hide"
    class="inline-flex items-center"
    :class="[color === 'normal' ? 'normal-link' : 'light-link']"
    :href="url"
    target="__BLANK"
  >
    {{ $t("common.learn-more") }}
    <ExternalLinkIcon class="w-4 h-4 ml-1" />
  </a>
</template>

<script lang="ts" setup>
import { ExternalLinkIcon } from "lucide-vue-next";
import { computed, type PropType } from "vue";
import { useActuatorV1Store } from "@/store";

const props = defineProps({
  url: {
    type: String,
    required: true,
  },
  color: {
    type: String as PropType<"normal" | "light">,
    default: "normal",
  },
  hideWhenEmbedded: {
    type: Boolean,
    default: false,
  },
});

const hide = computed(() => {
  if (!props.hideWhenEmbedded) return false;
  return useActuatorV1Store().appProfile.embedded;
});
</script>
