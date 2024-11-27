<template>
  <div class="flex items-center max-w-full overflow-hidden" v-bind="$attrs">
    <slot name="default">
      <span
        v-for="i in indent"
        :key="`indent-#${i}`"
        class="inline-block w-[20px] h-[20px] shrink-0 invisible"
        :data-indent="i"
      />
      <span
        v-if="!hideIcon"
        class="flex items-center justify-center shrink-0 w-[20px] h-[20px]"
      >
        <slot name="icon" />
      </span>
      <slot v-if="text" name="text">
        <HighlightLabelText
          v-if="highlight"
          :text="text"
          :keyword="keyword ?? ''"
          class="flex-1 truncate pl-[2px] min-w-[4rem]"
        />
        <span v-else class="pl-[2px]">{{ text }}</span>
      </slot>
      <slot v-else name="fallback-text" />
      <slot name="suffix" />
    </slot>
  </div>
</template>

<script setup lang="ts">
import HighlightLabelText from "./HighlightLabelText.vue";

defineProps<{
  text: string;
  indent?: number;
  keyword?: string;
  highlight?: boolean;
  hideIcon?: boolean;
}>();
</script>
