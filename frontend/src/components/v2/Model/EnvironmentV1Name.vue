<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && 'normal-link'"
  >
    <span>{{ environmentV1Name(environment) }}</span>
    <ProductionEnvironmentV1Icon
      :environment="environment"
      class="!text-current"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { Environment } from "@/types/proto/v1/environment_service";
import { environmentV1Name, environmentV1Slug } from "@/utils";
import ProductionEnvironmentV1Icon from "./ProductionEnvironmentV1Icon.vue";

const props = withDefaults(
  defineProps<{
    environment: Environment;
    tag?: string;
    link?: boolean;
  }>(),
  {
    tag: "span",
    link: true,
  }
);

const bindings = computed(() => {
  if (props.link) {
    return {
      to: `/environment/${environmentV1Slug(props.environment)}`,
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});
</script>
