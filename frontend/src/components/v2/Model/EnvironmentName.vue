<template>
  <component
    :is="link ? 'router-link' : tag"
    :to="link && `/environment/${environment.id}`"
    class="inline-flex items-center gap-x-1"
  >
    <span>{{ environmentName(environment) }}</span>
    <ProductionEnvironmentIcon :environment="environment" />
  </component>
</template>

<script lang="ts" setup>
import type { Environment, Instance } from "@/types";
import { environmentName } from "@/utils";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";

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

const onClick = (e: MouseEvent) => {
  if (props.link) {
    e.stopPropagation();
  }
};
</script>
