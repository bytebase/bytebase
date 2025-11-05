<template>
  <component :is="tag" class="flex items-center gap-x-0.5">
    <EngineIcon :engine="engine" />
    {{ title }}
    <span v-if="subtitle" class="text-xs! textinfolabel">
      {{ subtitle }}
    </span>
  </component>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { engineNameV1 } from "@/utils";

const RE_SUBTITLE = /\(.+?\)/;

const props = withDefaults(
  defineProps<{
    tag?: string;
    engine: Engine;
  }>(),
  {
    tag: "div",
  }
);

const name = computed(() => engineNameV1(props.engine));

const subtitle = computed(() => {
  const match = name.value.match(RE_SUBTITLE);
  if (!match) return "";
  return match[0];
});

const title = computed(() => {
  if (!subtitle.value) return name.value;
  return name.value.replace(subtitle.value, "").trim();
});
</script>
