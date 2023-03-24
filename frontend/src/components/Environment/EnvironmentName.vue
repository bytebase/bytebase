<template>
  <component :is="tag">
    {{ environmentName(environment) }}
    <ProductionEnvironmentIcon
      :environment="environment"
      class="!text-current"
    />
  </component>
</template>

<script lang="ts" setup>
import type { Environment } from "@/types";
import { environmentName } from "@/utils";
import ProductionEnvironmentIcon from "./ProductionEnvironmentIcon.vue";

withDefaults(
  defineProps<{
    environment: Environment;
    tag?: string;
  }>(),
  {
    tag: "span",
  }
);
</script>
