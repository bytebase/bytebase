<template>
  <component :is="tag">
    <span v-if="prefix" class="ml-1 text-gray-400">{{ prefix }}</span>
    {{ database.name }}
  </component>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Database } from "@/types";

const props = withDefaults(
  defineProps<{
    database: Database;
    tag?: string;
  }>(),
  {
    tag: "span",
  }
);

const prefix = computed(() => {
  const { database } = props;
  if (database.instance.engine === "REDIS") {
    return `${database.instance.name}`;
  }
  return "";
});
</script>
