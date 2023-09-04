<template>
  <NTooltip trigger="hover" :delay="500" :animated="false">
    <template #trigger>
      <NButton quaternary size="tiny" class="!px-1" v-bind="$attrs" @click="go">
        <heroicons-outline:external-link class="w-4 h-4" />
      </NButton>
    </template>
    {{ tooltip }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";

const props = defineProps<{
  link: string;
  tooltip: string;
}>();

const go = () => {
  window.open(props.link, "_blank");
};
</script>
