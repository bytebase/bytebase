<template>
  <component
    :is="link ? 'router-link' : tag"
    :to="link && `/environment/${environment.id}`"
    class="inline-flex items-center gap-x-1"
    @click.stop=""
  >
    <span>{{ environmentName(environment) }}</span>
    <ProductionEnvironmentIcon :environment="environment" />
  </component>
</template>

<script lang="ts" setup>
import type { Environment, Instance } from "@/types";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import { environmentName } from "@/utils";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";

withDefaults(
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
</script>
