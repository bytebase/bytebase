<template>
  <td class="table-cell text-sm leading-5 max-w-prose" :class="classNames">
    <slot />
  </td>
</template>

<script lang="ts" setup>
import { withDefaults, defineProps, computed } from "vue";

const props = withDefaults(
  defineProps<{
    leftPadding?: number;
    rightPadding?: number;
    compact?: boolean;
  }>(),
  {
    leftPadding: 2,
    rightPadding: 2,
    compact: false,
  }
);

const classNames = computed((): string[] => {
  const classNames: string[] = [];
  if (!props.compact) {
    classNames.push(`pl-${props.leftPadding}`);
    classNames.push(`pr-${props.rightPadding}`);
    classNames.push("py-2");
  }
  return classNames;
});
</script>
