<template>
  <InstanceEngineIcon :instance="instance" />
  <span class="text-sm -mr-0.5" :class="[!disabled && 'text-gray-500']">
    (
  </span>
  <ProtectedEnvironmentIcon
    :environment="instance.environment"
    class="w-4 h-4 text-inherit"
  />
  <span class="text-sm" :class="[!disabled && 'text-gray-500']">
    {{ instance.environment.name }}
  </span>
  <span class="text-sm -ml-0.5" :class="[!disabled && 'text-gray-500']">
    )
  </span>
</template>

<script setup lang="ts">
import type { Instance } from "@/types";

withDefaults(
  defineProps<{
    instance: Instance;
    disabled?: boolean;
  }>(),
  {
    disabled: false,
  }
);
</script>
