<template>
  <th
    scope="col"
    class="table-cell text-left text-xs font-medium text-gray-500 tracking-wider"
    :class="classNames"
  >
    <slot>{{ title }}</slot>
  </th>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  name: "BBTableHeaderCell",
  props: {
    title: {
      type: String,
      default: "",
    },
    leftPadding: {
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
      classNames.push("py-2");
    }
    return { classNames };
  },
});
</script>
