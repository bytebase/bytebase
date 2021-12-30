<template>
  <td class="table-cell text-sm leading-5 max-w-prose" :class="classNames">
    <slot />
  </td>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  name: "BBTableCell",
  props: {
    leftPadding: {
      type: Number,
      default: 2,
    },
    rightPadding: {
      type: Number,
      default: 2,
    },
    compact: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const classNames: string[] = [];
    if (!props.compact) {
      classNames.push(`pl-${props.leftPadding}`);
      classNames.push(`pr-${props.rightPadding}`);
      classNames.push("py-2");
    }
    return { classNames };
  },
});
</script>
