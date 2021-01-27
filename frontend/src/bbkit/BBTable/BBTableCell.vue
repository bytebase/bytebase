<template>
  <td class="table-cell px-4 py-2 whitespace-nowrap text-sm leading-5">
    <slot />
  </td>
</template>

<script lang="ts">
import { computed, reactive } from "vue";

export default {
  name: "BBTableCell",
  props: {},
  setup(props, ctx) {},
};
</script>
