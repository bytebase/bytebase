<template>
  <td
    class="table-cell text-sm leading-5 py-2 max-w-prose"
    :class="`pl-` + leftPadding + ` pr-` + rightPadding"
  >
    <slot />
  </td>
</template>

<script lang="ts">
export default {
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
  },
};
</script>
