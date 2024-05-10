<template>
  <NPopover
    :disabled="popoverDisabled"
    :placement="placement"
    style="max-height: 300px"
    width="trigger"
    scrollable
  >
    <div v-if="codeMode" :class="codeClass">
      <slot name="popover-header" />
      <highlight-code-block
        :code="displayPopoverContent"
        class="whitespace-pre-wrap"
      />
    </div>
    <div v-else class="whitespace-pre-wrap">
      {{ displayPopoverContent }}
    </div>

    <template #trigger>
      <slot name="default" :display-content="displayContent">
        <span :class="contentClass" v-bind="$attrs">{{ displayContent }}</span>
      </slot>
    </template>
  </NPopover>
</template>

<script lang="ts" setup>
import type { PopoverPlacement } from "naive-ui";
import { NPopover } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import type { VueClass } from "@/utils";

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
  codeClass: {
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

const popoverDisabled = computed(() => {
  return displayContent.value === props.content;
});
</script>
