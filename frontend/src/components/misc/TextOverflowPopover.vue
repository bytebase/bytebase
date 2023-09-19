<template>
  <NPopover
    :disabled="content.length <= maxLength"
    :placement="placement"
    style="max-height: 300px"
    width="trigger"
    scrollable
  >
    <highlight-code-block
      v-if="codeMode"
      :code="content"
      class="whitespace-pre-wrap"
    />
    <div v-else class="whitespace-pre-wrap">
      {{ content }}
    </div>

    <template #trigger>
      <span :class="contentClass">
        {{
          content.length <= maxLength
            ? content
            : content.substring(0, maxLength) + "..."
        }}
      </span>
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import { NPopover, PopoverPlacement } from "naive-ui";
import { PropType } from "vue";
import { VueClass } from "@/utils";

defineProps({
  content: {
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
  contentClass: {
    type: [String, Object, Array] as PropType<VueClass>,
    default: undefined,
  },
  codeMode: {
    type: Boolean,
    default: true,
  },
});
</script>
