<template>
  <InstanceV1EngineIcon :instance="instance" />
  <span class="text-sm -mr-0.5" :class="[!disabled && 'text-gray-500']">
    (
  </span>
  <ProductionEnvironmentV1Icon
    :environment="environment"
    class="w-4 h-4 text-inherit"
  />
  <span class="text-sm" :class="[!disabled && 'text-gray-500']">
    {{ environment.title }}
  </span>
  <span class="text-sm -ml-0.5" :class="[!disabled && 'text-gray-500']">
    )
  </span>
</template>

<script setup lang="ts">
import {
  InstanceV1EngineIcon,
  ProductionEnvironmentV1Icon,
} from "@/components/v2";
import { ComposedInstance } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";

withDefaults(
  defineProps<{
    instance: ComposedInstance;
    environment: Environment;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);
</script>
