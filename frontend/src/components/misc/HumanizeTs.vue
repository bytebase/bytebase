<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <span v-bind="$attrs">{{ humanized }}</span>
    </template>

    <span class="whitespace-nowrap">{{ detail }}</span>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { formatAbsoluteDateTime, humanizeTs } from "@/utils";

const props = defineProps({
  ts: {
    type: Number,
    required: true,
  },
});

const humanized = computed(() => humanizeTs(props.ts));

const detail = computed(() => formatAbsoluteDateTime(props.ts * 1000));
</script>
