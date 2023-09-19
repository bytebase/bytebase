<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    class="inline-flex items-center gap-x-1"
    :class="link && 'normal-link'"
  >
    <span>{{ environmentName(environment) }}</span>
    <ProductionEnvironmentIcon
      :environment="environment"
      class="!text-current"
    />
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";
import type { Environment } from "@/types";
import { environmentName, environmentSlug } from "@/utils";

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
      to: `/environment/${environmentSlug(props.environment)}`,
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
