<template>
  <component
    :is="link ? 'router-link' : tag"
    :to="link && `/instance/${instance.id}`"
    class="inline-flex items-center gap-x-1"
    @click.stop=""
  >
    <InstanceEngineIcon
      v-if="icon && iconPosition === 'prefix'"
      :instance="instance"
    />

    <span>{{ instanceName(instance) }}</span>

    <InstanceEngineIcon
      v-if="icon && iconPosition === 'suffix'"
      :instance="instance"
    />
  </component>
</template>

<script lang="ts" setup>
import type { Instance } from "@/types";
import InstanceEngineIcon from "@/components/InstanceEngineIcon.vue";
import { instanceName } from "@/utils";

withDefaults(
  defineProps<{
    instance: Instance;
    tag?: string;
    link?: boolean;
    icon?: boolean;
    iconPosition?: "prefix" | "suffix";
  }>(),
  {
    tag: "span",
    link: true,
    icon: true,
    iconPosition: "prefix",
  }
);
</script>
