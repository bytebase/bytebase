<template>
  <InstanceV1EngineIcon :instance="instance" />
  <span class="text-sm" :class="[!disabled && 'text-control-light']">
    {{ environment.title }}
  </span>
</template>

<script setup lang="ts">
import { InstanceV1EngineIcon } from "@/components/v2";
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
