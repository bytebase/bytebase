<template>
  <div class="flex items-center gap-x-1">
    <span v-if="showPrefix && resourcePrefix">{{ `${resourcePrefix}: ` }}</span>
    <component :is="resourceComponent" />
  </div>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { useResourceByName } from "./useResourceByName.ts";

const props = defineProps<{
  resource: string;
  link?: boolean;
  showPrefix?: boolean;
}>();

const { resourcePrefix, resourceComponent } = useResourceByName({
  resource: computed(() => props.resource ?? ""),
  link: props.link,
});
</script>
