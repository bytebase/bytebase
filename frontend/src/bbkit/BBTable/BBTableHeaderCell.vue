<template>
  <th
    scope="col"
    class="text-left text-xs font-medium text-gray-500 capitalize tracking-wider py-2"
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
  setup(props, ctx) {},
};
</script>
