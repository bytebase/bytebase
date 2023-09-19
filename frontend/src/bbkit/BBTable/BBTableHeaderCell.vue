<template>
  <th
    scope="col"
    class="table-cell text-left text-xs font-medium text-gray-500 tracking-wider"
    :class="classNames"
  >
    <slot>{{ title }}</slot>
  </th>
</template>

<script lang="ts" setup>
import { withDefaults, computed } from "vue";

const props = withDefaults(
  defineProps<{
    title?: string;
    leftPadding?: number;
    compact?: boolean;
  }>(),
  {
    title: "",
    leftPadding: 2,
    compact: false,
  }
);

const classNames = computed((): string[] => {
  const classNames: string[] = [];
  if (!props.compact) {
    classNames.push(`pl-${props.leftPadding}`);
    classNames.push("py-2");
  }
  return classNames;
});
</script>
