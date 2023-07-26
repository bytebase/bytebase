<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && !plain && 'normal-link'"
  >
    <span class="line-clamp-1">{{ environmentV1Name(environment) }}</span>
    <ProductionEnvironmentV1Icon
      :environment="environment"
      :class="iconClass ?? '!text-current'"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { Environment } from "@/types/proto/v1/environment_service";
import { VueClass, environmentV1Name, environmentV1Slug } from "@/utils";
import ProductionEnvironmentV1Icon from "./ProductionEnvironmentV1Icon.vue";

const props = withDefaults(
  defineProps<{
    environment: Environment;
    tag?: string;
    link?: boolean;
    plain?: boolean;
    iconClass?: VueClass;
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
    iconClass: undefined,
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
