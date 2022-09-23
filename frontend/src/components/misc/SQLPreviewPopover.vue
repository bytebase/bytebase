<template>
  <NPopover
    :disabled="statement.length <= maxLength"
    :placement="placement"
    style="max-height: 300px"
    width="trigger"
    scrollable
  >
    <highlight-code-block :code="statement" class="whitespace-pre-wrap" />

    <template #trigger>
      <span :class="statementClass">
        {{
          statement.length <= maxLength
            ? statement
            : statement.substring(0, maxLength) + "..."
        }}
      </span>
    </template>
  </NPopover>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NPopover, PopoverPlacement } from "naive-ui";
import { VueClass } from "@/utils";

export default defineComponent({
  name: "SQLPreviewPopover",
  components: { NPopover },
  props: {
    statement: {
      type: String,
      default: "",
    },
    maxLength: {
      type: Number,
      default: 100,
    },
    placement: {
      type: String as PropType<PopoverPlacement>,
      default: "top",
    },
    statementClass: {
      type: [String, Object, Array] as PropType<VueClass>,
      default: undefined,
    },
  },
});
</script>
