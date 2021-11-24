<template>
  <th
    scope="col"
    class="
      table-cell
      text-left text-xs
      font-medium
      text-gray-500
      tracking-wider
      py-2
    "
    :class="`pl-` + leftPadding"
  >
    {{ title }}
  </th>
</template>

<script lang="ts">
export default {
  name: "BBTableHeaderCell",
  props: {
    title: {
      required: true,
      type: String,
    },
    leftPadding: {
      type: Number,
      default: 2,
    },
  },
};
</script>
