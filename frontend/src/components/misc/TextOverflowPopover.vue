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
      :code="displayPopoverContent"
      class="whitespace-pre-wrap"
    />
    <div v-else class="whitespace-pre-wrap">
      {{ displayPopoverContent }}
    </div>

    <template #trigger>
      <span :class="contentClass">{{ displayContent }}</span>
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import { NPopover, PopoverPlacement } from "naive-ui";
import { PropType, computed } from "vue";
import { VueClass } from "@/utils";

const props = defineProps({
  content: {
    type: String,
    default: "",
  },
  maxLength: {
    type: Number,
    default: 100,
  },
  maxPopoverContentLength: {
    type: Number,
    default: 10000,
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
  lineWrap: {
    type: Boolean,
    default: true,
  },
  lineBreakReplacer: {
    type: String,
    default: "",
  },
});

const displayContent = computed(() => {
  const { content, lineWrap, maxLength } = props;
  if (lineWrap) return content;
  const displayContent = content.replace(/\n/g, props.lineBreakReplacer);
  return displayContent.length <= maxLength
    ? displayContent
    : displayContent.substring(0, maxLength) + "...";
});

const displayPopoverContent = computed(() => {
  const { content, maxPopoverContentLength } = props;
  return content.length <= maxPopoverContentLength
    ? content
    : content.substring(0, maxPopoverContentLength) + "...";
});
</script>
